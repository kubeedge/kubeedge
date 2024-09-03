# API
Schema of the external API types that are served by KubeEdge.

# Purpose
This library is the canonical location of the KubeEdge API definition. It will be published separately to avoid diamond dependency problems for users who depend on multiple KubeEdge components.
On the other hand, this library also provides available CRD informers/listers/clientsets and KubeEdge API documentation for users.

# Where does it come from?
api is synced from https://github.com/kubeedge/kubeedge/tree/master/staging/src/github.com/kubeedge/api.
Code changes are made in that location, merged into KubeEdge and later synced here.