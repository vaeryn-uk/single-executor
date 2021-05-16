package watchdog

type votes map[Id]bool

func createVotes(cluster Cluster) votes {
	v := make(votes)

	for node := range cluster.nodes {
		v[node] = false
	}

	return v
}

func (v votes) isMajority() bool {
	yes, no := 0, 0

	for _, voted := range v {
		if voted {
			yes++
		} else {
			no++
		}
	}

	return yes > no
}

func (v votes) reset() votes {
	for id, _ := range v {
		v[id] = false
	}

	return v
}

func (v votes) vote(id Id) votes {
	if _, ok := v[id]; ok {
		v[id] = true
	}

	return v
}
