
import models.io.upbound.awsm.elasticache.v1beta1 as elasticachev1beta1
import models.io.upbound.customer.v1alpha1 as customerv1alpha1
import models.k8s.apimachinery.pkg.apis.meta.v1 as metav1

oxr = option("params").oxr # observed composite resource
_ocds = option("params").ocds # observed composed resources
_dxr = option("params").dxr # desired composite resource
dcds = option("params").dcds # desired composed resources
_ctx = option("params").ctx or {} # pipeline context from previous functions

_metadata = lambda name: str -> any {
    { annotations = { "krm.kcl.dev/composition-resource-name" = name }}
}

# Get parameters from XR
_region = "us-east-1"
if oxr.spec?.parameters?.region:
    _region = oxr.spec.parameters.region

# Static cache-id label for matching users to usergroup
_cache_id = "prod-cache"

# Get discovered user IDs from pipeline context (set by usergroup-manager function)
_discovered_user_ids = []
if _ctx?.discoveredUserIDs:
    # Convert context list to KCL list
    _discovered_user_ids = [user for user in _ctx.discoveredUserIDs]

_items = [
    # ServerlessCache
    elasticachev1beta1.ServerlessCache {
        metadata: _metadata("serverless-cache")
        spec: {
            forProvider: {
                engine: "redis"
                region: _region
            }
        }
    }

    # UserGroup with dynamically discovered users from Go function
    # User IDs are discovered via AWS SDK by the usergroup-manager function
    # This includes the AWS built-in "default" user plus any custom users
    elasticachev1beta1.UserGroup {
        metadata: _metadata("user-group")
        spec: {
            forProvider: {
                engine: "redis"
                region: _region
                # Use discovered user IDs from context if available, otherwise empty
                userIds: _discovered_user_ids if _discovered_user_ids else []
            }
        }
    }
]
items = _items
