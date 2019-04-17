package types

// Store physical is and health status of a single GPU
type NvidiaGPUStatus struct {
	// Store the physical GPU id
	ID string `json:"id" protobuf:"bytes,1,opt,name=id"`
	// Health status
	Healthy bool `json:"healthy" protobuf:"varint,2,opt,name=healthy"`
}
