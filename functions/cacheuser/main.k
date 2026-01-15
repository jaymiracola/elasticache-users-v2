
import models.io.upbound.awsm.elasticache.v1beta1 as elasticachev1beta1
import models.io.upbound.customer.v1alpha1 as customerv1alpha1
import models.k8s.apimachinery.pkg.apis.meta.v1 as metav1

oxr = option("params").oxr # observed composite resource
_ocds = option("params").ocds # observed composed resources
_dxr = option("params").dxr # desired composite resource
dcds = option("params").dcds # desired composed resources

_metadata = lambda name: str -> any {
    { annotations = { "krm.kcl.dev/composition-resource-name" = name }}
}

# Get parameters from XR
_region = "us-east-1"
if oxr.spec?.parameters?.region:
    _region = oxr.spec.parameters.region

_username = "user"
if oxr.spec?.parameters?.username:
    _username = oxr.spec.parameters.username

# Static cache-id label for UserGroup matching
_cache_id = "prod-cache"

_items = [
    # User with limited access (on ~username:* +@all)
    elasticachev1beta1.User {
        metadata: {
            **_metadata("cache-user")
            # Label enables UserGroup selector to find this user
            labels: {
                "cache-id": _cache_id
            }
        }
        spec: {
            forProvider: {
                engine: "redis"
                userName: _username
                accessString: "on ~${_username}:* +@all"
                region: _region
                noPasswordRequired: True
            }
        }
    }
]
items = _items
