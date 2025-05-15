package responses

import (
	v1 "github.com/prometheus/prometheus/web/api/v1"
)

type PrometheusTargetsResponse struct {
	Clusters []*Cluster `json:"clusters"`
}

type Cluster struct {
	Name  string  `json:"name"`
	Nodes []*Node `json:"nodes"`
}

type Node struct {
	Name string `json:"name"`
	Pods []*Pod `json:"pods"`
}

type Pod struct {
	Name     string       `json:"name"`
	Response *v1.Response `json:"response"`
}

func (r *PrometheusTargetsResponse) findOrCreateCluster(clusterName string) *Cluster {
	for _, cluster := range r.Clusters {
		if cluster.Name == clusterName {
			return cluster
		}
	}

	r.Clusters = append(r.Clusters, &Cluster{
		Name:  clusterName,
		Nodes: []*Node{},
	})

	return r.Clusters[len(r.Clusters)-1]
}

func (r *PrometheusTargetsResponse) findOrCreateNode(cluster *Cluster, nodeName string) *Node {
	for _, node := range cluster.Nodes {
		if node.Name == nodeName {
			return node
		}
	}

	cluster.Nodes = append(cluster.Nodes, &Node{
		Name: nodeName,
		Pods: []*Pod{},
	})

	return cluster.Nodes[len(cluster.Nodes)-1]
}

func (r *PrometheusTargetsResponse) AddPodResponse(clusterName, nodeName, podName string, podResponse *v1.Response) {
	if r.Clusters == nil {
		r.Clusters = make([]*Cluster, 0)
	}

	cluster := r.findOrCreateCluster(clusterName)
	node := r.findOrCreateNode(cluster, nodeName)
	node.Pods = append(node.Pods, &Pod{
		Name:     podName,
		Response: podResponse,
	})
}
