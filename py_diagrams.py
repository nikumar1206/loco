from diagrams import Diagram, Cluster, Edge
from diagrams.onprem.compute import Server
from diagrams.k8s.compute import Pod
from diagrams.onprem.network import Envoy
from diagrams.onprem.database import PostgreSQL, ClickHouse
from diagrams.onprem.monitoring import Grafana
from diagrams.onprem.container import Docker

def create_loco_architecture():
    """Generate Loco architecture diagram using diagrams library"""
    
    # Graph attributes to control appearance and force left-to-right layout
    graph_attr = {
        "pad": "0.5",
        "nodesep": "0.4",
        "ranksep": "0.8",
        "splines": "spline",
        "concentrate": "false",
    }
    
    node_attr = {
        "fontsize": "12",
        "height": "0.6",
        "width": "0.6",
        "margin": "0.1",
    }
    
    edge_attr = {
        "minlen": "1",
    }
    
    # Create diagram with LR (left-to-right) direction
    with Diagram(
        "Loco Architecture", 
        filename="loco_architecture", 
        show=False, 
        direction="LR",
        graph_attr=graph_attr,
        node_attr=node_attr,
        edge_attr=edge_attr,
        outformat="png"
    ):
        
        # External components (outside K8s)
        users = Server("Users")
        nlb = Server("NLB")
        container_registry = Docker("Container Registry\n(GitLab/GitHub)")
        
        # Kubernetes Cluster
        with Cluster("Kubernetes Cluster"):
            
            # Envoy Gateway - positioned first to be at the top/entry point
            envoy = Envoy("L7 Envoy\nGateway")
            
            # Main row: loco-system and user namespaces side by side
            with Cluster("loco-system namespace", graph_attr={"bgcolor": "lightgreen"}):
                # loco-api pods without nested cluster
                api_pod1 = Pod("loco-api")
                api_pod2 = Pod("loco-api")
                api_pod3 = Pod("loco-api")
                
                db = PostgreSQL("PostgreSQL")
                
                # Connect pods to database within namespace
                api_pod1 >> db
                api_pod2 >> db
                api_pod3 >> db
            
            # User Namespaces - keep services compact
            with Cluster("User Namespaces", graph_attr={"bgcolor": "lightyellow"}):
                svc_a = Pod("Service A")
                svc_b = Pod("Service B")
                svc_c = Pod("Service C")
            
            # Observability namespace - horizontal layout to span width
            with Cluster("observability namespace", graph_attr={"bgcolor": "lightblue", "style": "filled", "rankdir": "LR"}):
                otel = Pod("otel-collector")
                clickhouse = ClickHouse("ClickHouse")
                grafana = Grafana("Grafana")
                
                # Observability chain (horizontal)
                otel >> clickhouse >> grafana
        
        # Main flow connections
        users >> nlb >> envoy
        
        # Envoy to loco-api pods
        envoy >> api_pod1
        envoy >> api_pod2
        envoy >> api_pod3
        
        # loco-api pods to user services
        api_pod1 >> svc_a
        api_pod2 >> svc_b
        api_pod3 >> svc_c
        
        # Services to otel-collector
        svc_a >> otel
        svc_b >> otel
        svc_c >> otel
        
        # Container registry connection (dashed)
        svc_a >> Edge(style="dashed", color="gray") >> container_registry

if __name__ == "__main__":
    create_loco_architecture()
    print("✓ Loco architecture diagram generated: loco_architecture.png")