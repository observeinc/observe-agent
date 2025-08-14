package agentresourceextension

type AgentResourceProvider interface {
	GetAgentInstanceId() string
	GetAgentStartTime() int64
	GetAgentData() AgentLocalData
}