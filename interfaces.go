package lidario

// LidarFile defines the common interface for both LAS and LAZ files
type LidarFile interface {
	// LasPoint reads a point at the specified index
	LasPoint(pointIndex int) (LasPointer, error)
	
	// GetXYZ gets the coordinates of a specific point
	GetXYZ(pointIndex int) (float64, float64, float64, error)
	
	// Close closes the file
	Close() error
	
	// GetHeader returns the header information
	GetHeader() *LasHeader
	
	// GetPointCount returns the total number of points
	GetPointCount() uint32
	
	// IsCompressed returns true if this is a compressed file
	IsCompressed() bool
}

// Ensure LasFile implements LidarFile interface
var _ LidarFile = (*LasFile)(nil)

// GetHeader returns the header for LasFile (implement interface)
func (lf *LasFile) GetHeader() *LasHeader {
	return &lf.Header
}

// GetPointCount returns the point count for LasFile (implement interface)
func (lf *LasFile) GetPointCount() uint32 {
	return uint32(lf.Header.NumberPoints)
}

// IsCompressed returns false for uncompressed LAS files
func (lf *LasFile) IsCompressed() bool {
	return false
}