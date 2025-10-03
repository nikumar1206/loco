package klogmux

import (
	"bufio"
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// LogEntry represents a single log line with metadata
type LogEntry struct {
	Timestamp time.Time
	Namespace string
	PodName   string
	Container string
	Message   string
	IsError   bool
}

// FilterFunc is a function that filters log entries
type FilterFunc func(entry LogEntry) bool

// TransformFunc transforms a log entry
type TransformFunc func(entry LogEntry) LogEntry

// LogStream represents a multiplexed log stream
type LogStream struct {
	// Immutable configuration (set at creation, never modified)
	client     kubernetes.Interface
	namespace  string
	selector   labels.Selector
	containers []string
	filters    []FilterFunc
	transforms []TransformFunc
	follow     bool
	tailLines  *int64
	since      *metav1.Time
	timestamps bool
	previous   bool

	// Concurrency primitives (already thread-safe)
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	entries chan LogEntry
	errors  chan error

	// Mutable state (protected by respective mechanisms)
	running atomic.Bool // Atomic - no lock needed
	pods    sync.Map    // Thread-safe map - no lock needed
}

// podStream represents logs from a single pod container
type podStream struct {
	namespace string
	name      string
	container string
	cancel    context.CancelFunc
}

// Builder constructs a LogStream with a fluent interface
type Builder struct {
	client     kubernetes.Interface
	namespace  string
	selector   labels.Selector
	containers []string
	filters    []FilterFunc
	transforms []TransformFunc
	follow     bool
	tailLines  *int64
	since      *metav1.Time
	timestamps bool
	previous   bool
	bufferSize int
}

// NewBuilder creates a new LogStream builder
func NewBuilder(client kubernetes.Interface) *Builder {
	return &Builder{
		client:     client,
		namespace:  "",
		containers: []string{},
		filters:    []FilterFunc{},
		transforms: []TransformFunc{},
		follow:     false,
		timestamps: false,
		previous:   false,
		bufferSize: 1000,
	}
}

// Namespace sets the namespace to query
func (b *Builder) Namespace(namespace string) *Builder {
	b.namespace = namespace
	return b
}

// AllNamespaces queries all namespaces
func (b *Builder) AllNamespaces() *Builder {
	b.namespace = ""
	return b
}

// LabelSelector sets the label selector for pods
func (b *Builder) LabelSelector(selector string) *Builder {
	if selector != "" {
		parsed, err := labels.Parse(selector)
		if err == nil {
			b.selector = parsed
		}
	}
	return b
}

// PodSelector adds a pod name filter
func (b *Builder) PodSelector(pattern string) *Builder {
	if pattern != "" {
		re, err := regexp.Compile(pattern)
		if err == nil {
			b.filters = append(b.filters, func(entry LogEntry) bool {
				return re.MatchString(entry.PodName)
			})
		}
	}
	return b
}

// ContainerSelector sets specific containers to watch
func (b *Builder) ContainerSelector(containers ...string) *Builder {
	b.containers = append(b.containers, containers...)
	return b
}

// ContainerPattern adds a container name pattern filter
func (b *Builder) ContainerPattern(pattern string) *Builder {
	if pattern != "" {
		re, err := regexp.Compile(pattern)
		if err == nil {
			b.filters = append(b.filters, func(entry LogEntry) bool {
				return re.MatchString(entry.Container)
			})
		}
	}
	return b
}

// Follow enables following log streams
func (b *Builder) Follow(follow bool) *Builder {
	b.follow = follow
	return b
}

// TailLines sets the number of lines to tail from the end
func (b *Builder) TailLines(lines int64) *Builder {
	b.tailLines = &lines
	return b
}

// Since filters logs to those after a specific time
func (b *Builder) Since(duration time.Duration) *Builder {
	t := metav1.NewTime(time.Now().Add(-duration))
	b.since = &t
	return b
}

// Previous gets previous container logs
func (b *Builder) Previous(previous bool) *Builder {
	b.previous = previous
	return b
}

// Timestamps includes timestamps in log output
func (b *Builder) Timestamps(timestamps bool) *Builder {
	b.timestamps = timestamps
	return b
}

// Filter adds a custom filter function
func (b *Builder) Filter(filter FilterFunc) *Builder {
	b.filters = append(b.filters, filter)
	return b
}

// MessageFilter filters by message content
func (b *Builder) MessageFilter(pattern string) *Builder {
	if pattern != "" {
		re, err := regexp.Compile(pattern)
		if err == nil {
			b.filters = append(b.filters, func(entry LogEntry) bool {
				return re.MatchString(entry.Message)
			})
		}
	}
	return b
}

// ExcludePattern excludes messages matching pattern
func (b *Builder) ExcludePattern(pattern string) *Builder {
	if pattern != "" {
		re, err := regexp.Compile(pattern)
		if err == nil {
			b.filters = append(b.filters, func(entry LogEntry) bool {
				return !re.MatchString(entry.Message)
			})
		}
	}
	return b
}

// Transform adds a transformation function
func (b *Builder) Transform(transform TransformFunc) *Builder {
	b.transforms = append(b.transforms, transform)
	return b
}

// BufferSize sets the channel buffer size
func (b *Builder) BufferSize(size int) *Builder {
	b.bufferSize = size
	return b
}

// Build creates the LogStream
func (b *Builder) Build() *LogStream {
	return &LogStream{
		client:     b.client,
		namespace:  b.namespace,
		selector:   b.selector,
		containers: b.containers,
		filters:    b.filters,
		transforms: b.transforms,
		follow:     b.follow,
		tailLines:  b.tailLines,
		since:      b.since,
		timestamps: b.timestamps,
		previous:   b.previous,
		entries:    make(chan LogEntry, b.bufferSize),
		errors:     make(chan error, 100),
	}
}

// Start begins streaming logs from matching pods
// Thread-safe: can be called multiple times, but only the first call succeeds
func (s *LogStream) Start(ctx context.Context) error {
	// Atomic check-and-set - prevents double start
	if !s.running.CompareAndSwap(false, true) {
		return fmt.Errorf("stream already running")
	}

	s.ctx, s.cancel = context.WithCancel(ctx)

	factory := informers.NewSharedInformerFactoryWithOptions(
		s.client,
		30*time.Second,
		informers.WithNamespace(s.namespace),
		informers.WithTweakListOptions(func(opts *metav1.ListOptions) {
			if s.selector != nil {
				opts.LabelSelector = s.selector.String()
			}
			opts.FieldSelector = fields.Everything().String()
		}))

	informer := factory.Core().V1().Pods().Informer()
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			pod := obj.(*corev1.Pod)
			s.handlePod(pod)
		},
		UpdateFunc: func(oldObj, newObj any) {
			pod := newObj.(*corev1.Pod)
			s.handlePod(pod)
		},
		DeleteFunc: func(obj any) {
			pod := obj.(*corev1.Pod)
			s.removePod(pod)
		},
	})

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		informer.Run(s.ctx.Done())
	}()

	return nil
}

// Stop gracefully stops the log stream
// Thread-safe: can be called multiple times safely
func (s *LogStream) Stop() {
	// Atomic check-and-set - prevents double stop
	if !s.running.CompareAndSwap(true, false) {
		return
	}

	// Signal all goroutines to stop
	s.cancel()

	// Wait for all goroutines to finish
	s.wg.Wait()

	// Safe to close channels - no more writers
	close(s.entries)
	close(s.errors)
}

// Entries returns the channel for receiving log entries
func (s *LogStream) Entries() <-chan LogEntry {
	return s.entries
}

// Errors returns the error channel
func (s *LogStream) Errors() <-chan error {
	return s.errors
}

// handlePod starts log streams for a pod's containers
// Called by informer callbacks (potentially concurrent)
func (s *LogStream) handlePod(pod *corev1.Pod) {
	// Only stream running pods (or previous containers if requested)
	if pod.Status.Phase != corev1.PodRunning && !s.previous {
		return
	}

	containers := s.getContainers(pod)

	// Start streams for each container
	// LoadOrStore is atomic and thread-safe
	for _, c := range containers {
		key := pod.Namespace + "/" + pod.Name + "/" + c

		// Attempt to reserve this stream slot
		// If already exists (loaded=true), skip
		// If new (loaded=false), start the stream
		if _, loaded := s.pods.LoadOrStore(key, nil); !loaded {
			s.startPodStream(pod, c, key)
		}
	}
}

// removePod stops and removes log streams for a pod
// Called by informer delete callback
func (s *LogStream) removePod(pod *corev1.Pod) {
	containers := s.getContainers(pod)

	for _, c := range containers {
		key := pod.Namespace + "/" + pod.Name + "/" + c

		// Atomically remove and get the stream
		if val, ok := s.pods.LoadAndDelete(key); ok {
			if ps := val.(*podStream); ps != nil {
				ps.cancel() // Stop the stream goroutine
			}
		}
	}
}

// getContainers returns the list of containers to stream for a pod
func (s *LogStream) getContainers(pod *corev1.Pod) []string {
	// If specific containers were requested, use those
	if len(s.containers) > 0 {
		return s.containers
	}

	// Otherwise, stream all containers in the pod
	containers := make([]string, 0, len(pod.Spec.Containers))
	for _, c := range pod.Spec.Containers {
		containers = append(containers, c.Name)
	}
	return containers
}

// startPodStream initializes and starts streaming logs from a pod container
func (s *LogStream) startPodStream(pod *corev1.Pod, container, key string) {
	ctx, cancel := context.WithCancel(s.ctx)

	stream := &podStream{
		namespace: pod.Namespace,
		name:      pod.Name,
		container: container,
		cancel:    cancel,
	}

	// Replace the nil placeholder with the actual stream
	s.pods.Store(key, stream)

	// Start the streaming goroutine
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.streamPodLogs(ctx, stream)
	}()
}

// streamPodLogs streams logs from a single pod container
// Runs in its own goroutine
func (s *LogStream) streamPodLogs(ctx context.Context, stream *podStream) {
	opts := &corev1.PodLogOptions{
		Container:  stream.container,
		Follow:     s.follow,
		Previous:   s.previous,
		Timestamps: s.timestamps,
	}

	if s.tailLines != nil {
		opts.TailLines = s.tailLines
	}

	if s.since != nil {
		opts.SinceTime = s.since
	}

	req := s.client.CoreV1().Pods(stream.namespace).GetLogs(stream.name, opts)

	reader, err := req.Stream(ctx)
	if err != nil {
		select {
		case s.errors <- fmt.Errorf("failed to stream logs for %s/%s/%s: %w",
			stream.namespace, stream.name, stream.container, err):
		default:
			// Error channel full, drop error
		}
		return
	}
	defer reader.Close()

	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 2048), 1024*1024)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
			line := scanner.Text()
			entry := s.parseLogEntry(stream, line)

			if s.shouldInclude(entry) {
				entry = s.applyTransforms(entry)

				select {
				case s.entries <- entry:
				case <-ctx.Done():
					return
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		select {
		case s.errors <- fmt.Errorf("scanner error for %s/%s/%s: %w",
			stream.namespace, stream.name, stream.container, err):
		default:
			// Error channel full, drop error
		}
	}
}

// parseLogEntry parses a log line into a LogEntry
func (s *LogStream) parseLogEntry(stream *podStream, line string) LogEntry {
	entry := LogEntry{
		Namespace: stream.namespace,
		PodName:   stream.name,
		Container: stream.container,
		Message:   line,
	}

	// Parse timestamp if present
	if s.timestamps && len(line) > 30 {
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 {
			if t, err := time.Parse(time.RFC3339Nano, parts[0]); err == nil {
				entry.Timestamp = t
				entry.Message = parts[1]
			}
		}
	}

	// Simple error detection (case-insensitive)
	lower := strings.ToLower(entry.Message)
	if strings.Contains(lower, "error") ||
		strings.Contains(lower, "fatal") ||
		strings.Contains(lower, "panic") {
		entry.IsError = true
	}

	return entry
}

// shouldInclude checks if entry passes all filters
func (s *LogStream) shouldInclude(entry LogEntry) bool {
	for _, filter := range s.filters {
		if !filter(entry) {
			return false
		}
	}
	return true
}

// applyTransforms applies all transforms to an entry
func (s *LogStream) applyTransforms(entry LogEntry) LogEntry {
	for _, transform := range s.transforms {
		entry = transform(entry)
	}
	return entry
}

// cleanupStreams stops all pod streams
func (s *LogStream) cleanupStreams() {
	s.pods.Range(func(key, value any) bool {
		if ps := value.(*podStream); ps != nil {
			ps.cancel()
		}
		s.pods.Delete(key)
		return true // continue iteration
	})
}

// Convenience functions for common use cases

// StreamAll creates a simple stream for all pods in a namespace
func StreamAll(ctx context.Context, client kubernetes.Interface, namespace string) (*LogStream, error) {
	stream := NewBuilder(client).
		Namespace(namespace).
		Follow(true).
		Build()

	if err := stream.Start(ctx); err != nil {
		return nil, err
	}

	return stream, nil
}

// StreamWithLabel creates a stream for pods with specific label
func StreamWithLabel(ctx context.Context, client kubernetes.Interface, namespace, labelSelector string) (*LogStream, error) {
	stream := NewBuilder(client).
		Namespace(namespace).
		LabelSelector(labelSelector).
		Follow(true).
		Build()

	if err := stream.Start(ctx); err != nil {
		return nil, err
	}

	return stream, nil
}
