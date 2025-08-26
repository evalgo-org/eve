package common

type FlowContainer struct {
	Image string `json:"container"`
	Tag   string `json:"container_tag"`
}

type FlowDocument struct {
	Version   string        `json:"version"`
	Process   string        `json:"process"`
	Container FlowContainer `json:"container"`
}

type FlowMessage struct {
	Version string `json:"version"`
	DocID   string `json:"doc_id"`
	DocRev  string `json:"doc_rev"`
}
