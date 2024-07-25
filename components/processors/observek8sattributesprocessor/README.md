# Observe K8s Attributes Processor
This processor operates on K8s resource logs from the `k8sobjectsreceiver` and adds additional attributes. 


## Caveats
This processor currently expects the `kind` field to be set at the base level of the event. In the case of `watch` events from the `k8sobjectsreceiver`, this field is instead present inside of the `object` field. This processor currently expects this field to be lifted from inside the `object` field to the base level by a transform processor earlier in the pipeline. If that isn't set up, this processor will only calculate status for `pull` events from the `k8sobjectsreceiver`.

## Emitted Attributes

| Attribute Key                     | Description                                                  |
|-----------------------------------|--------------------------------------------------------------|
| `observe_transform.facets.status` | The derived Pod status based on the current Pod description. |