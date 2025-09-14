# Technical Design Document: Docker Image Vulnerability Scanning for Loco

**Disclaimer** : Written by GPT-5-Mini and I.

## **1. Goals**

* Ensure Docker images deployed through `loco deploy` are scanned for vulnerabilities.
* Block deployments with **HIGH** or **CRITICAL** vulnerabilities.
* Provide developers **fast feedback** if local Trivy is available (optional).
* Avoid deploying images that may have vulnerabilities in the packages installed or in the kernel.

---

## **2. System Architecture**

### **2.1 Components**

| Component                      | Role                                                                                         |
| ------------------------------ | -------------------------------------------------------------------------------------------- |
| **Client CLI (`loco deploy`)** | Optional pre-scan if Trivy exists. Provides warning to developer. Nice to have, not required.|
| **Server/Deployment API**      | Centralized scanning pipeline. Pulls the pushed image and runs Trivy scan before deployment. |
| **Registry**                   | Stores built images. The server pulls from here for scanning.                                |
| **Trivy**                      | Vulnerability scanner. Installed on server (as container image or binary).                   |

---

### **2.2 Flow**

**Client CLI:**

1. Build Docker image locally during loco deploy via the docker-go client.
2. Optional: run Trivy scan if binary exists.
   * Output warning for HIGH/CRITICAL vulnerabilities.
   * Blocks pushing to registry.
3. Push image to private registry if build is successful.

**Server Deployment API:**
0. Include trivy binary in loco-api server.
1. Pull image from registry.
2. Run Trivy scan on the image. We can just shell out to the binary.
3. Evaluate results:
   * Fail deployment if HIGH/CRITICAL vulnerabilities exist.
   * Otherwise, continue to create Kubernetes deployment.
4. (Nice to have!) Store scan event into DB for auditing purposes.

**Diagram (simplified)**

```
[Developer Machine]
    |
    | docker build
    | optional trivy pre-scan (shell-out)
    V
[Registry] <--- push image
    |
    | server pulls image
    V
[Server Trivy Scan]
    |-- fail if HIGH/CRITICAL
    |-- else deploy to K8s
    V
[Deploy on Cluster]
```

---

## **3. Server-Side Implementation Plan**

### **3.1 Trivy Setup**

* Install Trivy on server:

  * Option A: Sidecar container in deployment job pod.
  * Option B: Binary in a scanner worker image.
* Ensure network access to the registry.

### **3.2 API Changes**

* Extend `DeployApp` function to:

  1. Pull image from registry (will require access to Gitlab Creds).
  2. Run Trivy scan: `trivy image --quiet --format json <image>`
    - simple shell out.
  3. Parse JSON output and check for HIGH/CRITICAL vulnerabilities.
  4. Return error to client if deployment blocked.

### **3.3 Data Handling**

* Optional: save scan results to server DB (for audit, reports, or dashboard).
* Include fields: `image`, `vulnerability ID`, `severity`, `package`, `installed version`.

### **3.4 Error Handling**

* Deployment fails if any HIGH or CRITICAL vulnerabilities exist.
* Other severities are logged/warned.
    - (Nice to have) Report severities back to the user with CVV links / mitigation steps?
* Server API returns JSON with scan summary if blocked.

---

## **4. Client-Side Optional Pre-Scan**

### **4.1 Rationale**

* Provides fast developer feedback.
* Cannot enforce deployment rules (just advisory).

### **4.2 Implementation**

* Detect Trivy binary: `exec.LookPath("trivy")`.
* Run scan: `trivy image --quiet --format json <image>`
* Block deployment on HIGH/CRITICAL issues and just warnings on others.

### **4.3 CLI Step Example**

```go
{
    Title: "Optional local scan (Trivy)",
    Run: func(logf func(string)) error {
        path, err := exec.LookPath("trivy")
        if err != nil {
            logf("Trivy not found locally; skipping pre-scan")
            return nil
        }

        result, err := scanner.ScanImage(dockerCli.ImageName, path)
        if err != nil {
            logf(fmt.Sprintf("local trivy scan failed: %v", err))
            return nil
        }

        for _, r := range result.Results {
            for _, v := range r.Vulnerabilities {
                if v.Severity == "HIGH" || v.Severity == "CRITICAL" {
                    logf(fmt.Sprintf("⚠️ local vulnerability: %s in %s (%s)", v.Severity, v.PkgName, v.VulnerabilityID))
                }
            }
        }
        return nil
    },
}
```

---

## **5. Deployment Pipeline Integration**

**Server Scan Step (Mandatory)**:

* Insert immediately after image push in your CI/CD pipeline or `apiClient.DeployApp`.
* Fail pipeline on HIGH/CRITICAL vulnerabilities.
* Optional: asynchronous reporting to dashboard.

**Client Pre-Scan Step (Optional)**:

* Insert after local Docker build.
* Advisory only.
* Skip if Trivy not installed.

---

## **6. Security Considerations**

* **Registry authentication**: ensure server can pull private images securely.
* **Server trust**: all scans must run in trusted server environment to avoid bypass.
* **Resource limits**: Trivy scans can be memory-intensive, use CPU/memory limits in server pods.

---

## **7. Optional Enhancements**

* Support **CVSS thresholds** configurable by project or team.
    - Threshholds may only be more strict than Loco's. Loco's thresholds cannot be bypassed.
* Cache Trivy DB on server side to speed up repeated scans.
* Add scheduled scans for already deployed images.
    - In this case, do we want to potentially remove the affected deployment after some message back to the user.
* Store historical vulnerabilities per image/tag.
* Scanning and deployment, may need to move to a separate service, to avoid blowing up the loco-server.
    - Deploying an app should theoretically be an async process.
    - Can be CPU/Memory intensive.
    - Better for separation of concerns, dep-management as rest of loco-api does not need the trivy binary.
