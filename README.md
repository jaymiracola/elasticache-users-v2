  # ElastiCache Users v2

  Dynamic AWS ElastiCache UserGroup management using Crossplane v2 composition functions.

  ## Problem

  Crossplane's label selectors can't dynamically discover managed resources across different namespaces when using namespace-scoped resources. This makes it difficult to maintain an ElastiCache UserGroup that includes users created in different namespaces.

  ## Solution

  This configuration uses a **Go composition function** that queries the AWS ElastiCache API directly to discover all users, then passes them through the pipeline context to a KCL function that creates the UserGroup.
