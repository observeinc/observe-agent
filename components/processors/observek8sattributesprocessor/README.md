# Observe K8s Attributes Processor
This processor operates on K8s resource logs from the `k8sobjectsreceiver` and adds additional attributes. 

## Purpose

This is a specialized processor component that enriches Kubernetes resource logs with additional attributes.  The processor essentially takes raw Kubernetes events and adds computed, normalized attributes that make the data more useful for observability purposes. 

The processor is designed to be part of a larger observability pipeline, enhancing Kubernetes resource logs with additional context and derived attributes that make the logs more useful for monitoring and debugging purposes.

This processor plays a crucial role in enriching Kubernetes observability data by adding computed attributes and status information that might not be directly available from the raw Kubernetes API responses.

## Technical Rationale

This processor shares much of its logic with `kubectl`, the official Kubernetes command-line tool. When you run `kubectl describe <resource>`, kubectl transforms raw Kubernetes API data into human-readable information. Our processor uses the same transformation logic to generate its facets. This means most of the attributes you see in Observe's explorer are computed using the same battle-tested code that powers `kubectl`, ensuring consistency and reliability in how Kubernetes resource states are interpreted and displayed.

Another key reason to create a dedicated processor is to compute attributes that cannot be computed with OTTL (the transformation language used by the Opentelemetry’s transform processor). One of the main limitations of that language is that it cannot iterate over lists/maps. With the custom processor we can access those elements in Go as structured objects, and leverage the power of the programming language to be able to access and manipulate data contained in such objects.

Beyond adding facets, the processor performs another critical function: automatic secret redaction. While preserving the raw event structure and secret names, the processor replaces any secret values with "REDACTED" before ingestion. This security feature protects customers from accidentally exposing sensitive information, even if their Kubernetes clusters inadvertently leak secrets in their events. By performing redaction at the processor level, we ensure that secret values never reach our storage system.

## Caveats
> [!CAUTION]
> This processor currently expects the `kind` field to be set at the base level of the event. In the case of `watch` events from the `k8sobjectsreceiver`, this field is instead present inside of the `object` field. This processor currently expects this field to be lifted from inside the `object` field to the base level by a transform processor earlier in the pipeline. If that isn't set up, this processor will only calculate status for `pull` events from the `k8sobjectsreceiver`.

## Description

1. **Main Purpose**:
    - Processes logs from the **`k8sobjectsreceiver`** (a Kubernetes objects receiver)
    - Adds additional attributes and metadata to various Kubernetes resource types
    - Calculates and derives status information for different Kubernetes objects
2. **Supported Kubernetes Resources**: The processor handles multiple Kubernetes resource types including:
    - Core Resources: Pods, Nodes, Services, ServiceAccounts, Endpoints, ConfigMaps, Secrets
    - Apps: StatefulSets, DaemonSets, Deployments
    - Workloads: Jobs, CronJobs
    - Storage: PersistentVolumes, PersistentVolumeClaims
    - Network: Ingress
3. **Key Features**:
    - Calculates derived status information for various resources
    - Adds metadata and attributes based on resource type
    - Processes both "watch" and "pull" events from the Kubernetes API
    - Handles resource body transformations and attribute enrichment
4. **Specific Actions Per Resource**: Each resource type has its own set of actions that add specific attributes:
    - Pods: Status, container counts, readiness state, and conditions
    - Nodes: Status, roles, and node pool information
    - Services: Load balancer ingress, selectors, ports, and external IPs
    - Jobs: Status and duration calculations
    - And many more resource-specific attributes
5. **Important Note**: There's a caveat that the processor expects the **`kind`** field to be at the base level of the event. For watch events, this field needs to be lifted from the **`object`** field to the base level by a transform processor earlier in the pipeline.

## Example

**Input Example (Kubernetes Pod Event)**:

```json
{
   "apiVersion": "v1",
   "kind": "Pod",
   "metadata": {      "name": "purge-old-datasets-28688813-sspnn",
      "namespace": "eng",
      "labels": {
         "observeinc.com/app": "apiserver",
         "observeinc.com/environment": "eng"
      }
   },
   "status": {
      "containerStatuses": [
         {
            "ready": true,
            "restartCount": 2,
            "state": {"running": {...}}
         },
         {
            "ready": true,
            "restartCount": 3,
            "state": {"running": {...}}
         },
         {
            "ready": false,
            "restartCount": 0,
            "state": {"waiting": {...}}
         }
      ],
      "conditions": [
         // Various pod conditions...
      ]
   }
}
```

**Output (Added OTEL Attributes)**:

```json
{
   // Original event data remains unchanged, but new attributes are added:
   "attributes": {
      "observe_transform": {
         "facets": {
            // Derived status based on pod conditions and state
            "status": "Terminating",

            // Container statistics
            "total_containers": 4,
            "ready_containers": 3,
            "restarts": 5,

            // Pod conditions as a map
            "conditions": {
               "PodScheduled": true,
               "Initialized": true,
               "Ready": false,
               "ContainersReady": false,
               "PodHasNetwork": true
            },

            // If pod has readiness gates
            "readinessGatesReady": 1,
            "readinessGatesTotal": 2
         }
      }
   }
}
```

## Implementation Details

The processor enriches the original Kubernetes event by:

1. **Computing Status**: It analyzes the pod's conditions, container states, and metadata to determine a high-level status (e.g., "Terminating")
2. **Container Statistics**: It calculates:
    - Total number of containers of a Pod
    - Number of ready containers of a Pod
    - Total restart count across all containers of a Pod
3. **Condition Analysis**: It transforms the pod's conditions into an easily queryable map
4. **Readiness Information**: If the pod has readiness gates, it computes how many are ready vs total

Similar transformations happen for other Kubernetes resources. For example:

- For **Services**: It adds load balancer status, external IPs, and port information
- For **Nodes**: It adds derived roles, pool information, and overall status
- For **Jobs**: It adds duration and completion status
- For **Ingress**: It adds routing and backend service information

These enriched attributes make it much easier to:

1. Query and filter Kubernetes resources based on their state
2. Create meaningful visualizations and dashboards
3. Set up monitoring and alerting based on derived states
4. Analyze the health and status of your Kubernetes resources

## Added Attributes

The processor adds a list of attributes under the `observe_transform.facets` namespace. The processor computes these attributes based on the raw Kubernetes resource state and adds them to make querying and monitoring easier. Note that some attributes might be conditionally present based on the resource state. For example, load balancer information will only be present for Services of type LoadBalancer, and certain status attributes will only appear when specific conditions are met.

### **Pod Attributes**

- **`observe_transform.facets.status`** - Overall pod status
- **`observe_transform.facets.total_containers`** - Total number of containers
- **`observe_transform.facets.ready_containers`** - Number of ready containers
- **`observe_transform.facets.restarts`** - Total restart count
- **`observe_transform.facets.readinessGatesReady`** - Number of ready readiness gates
- **`observe_transform.facets.readinessGatesTotal`** - Total number of readiness gates
- **`observe_transform.facets.conditions`** - Map of pod conditions
- **`observe_transform.facets.cronjob_name`** - Name of parent CronJob if applicable
- **`observe_transform.facets.statefulset_name`** - Name of parent StatefulSet if applicable
- **`observe_transform.facets.daemonset_name`** - Name of parent DaemonSet if applicable

### **Node Attributes**

- **`observe_transform.facets.status`** - Node status
- **`observe_transform.facets.roles`** - Node roles (e.g., master, worker)
- **`observe_transform.facets.pool`** - Node pool information (for managed K8s services)

### **Service Attributes**

- **`observe_transform.facets.loadBalancerIngress`** - Load balancer ingress information
- **`observe_transform.facets.selector`** - Service selector labels
- **`observe_transform.facets.ports`** - Service ports configuration
- **`observe_transform.facets.externalIPs`** - External IPs assigned to the service

### **Job Attributes**

- **`observe_transform.facets.status`** - Job status
- **`observe_transform.facets.duration`** - Job duration

### **ServiceAccount Attributes**

- **`observe_transform.facets.secrets`** - Associated secrets
- **`observe_transform.facets.secretsNames`** - Names of associated secrets
- **`observe_transform.facets.imagePullSecrets`** - Image pull secrets

### **Endpoints Attributes**

- **`observe_transform.facets.endpoints`** - List of endpoints

### **ConfigMap Attributes**

- **`observe_transform.facets.data`** - ConfigMap data

### **StatefulSet Attributes**

- **`observe_transform.facets.selector`** - StatefulSet selector labels

### **Deployment Attributes**

- **`observe_transform.facets.selector`** - Deployment selector labels

### **Ingress Attributes**

- **`observe_transform.facets.loadBalancer`** - Load balancer information

### **PersistentVolume and PersistentVolumeClaim Attributes**

- Various storage-related attributes (specific attributes depend on the storage provider)