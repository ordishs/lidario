package lidario

import (
	"errors"
	"fmt"
	"sync"
)

// LazFile represents a LAZ file with embedded LAS functionality
type LazFile struct {
	fileName     string
	fileMode     string
	reader       *LaszipReader
	Header       LasHeader
	VlrData      []VLR
	geokeys      GeoKeys
	isCompressed bool
	currentPoint int
	sync.RWMutex
}

// NewLazFile creates a new LazFile for reading compressed LAZ files
func NewLazFile(fileName, fileMode string) (*LazFile, error) {
	if fileMode != "r" && fileMode != "rh" {
		return nil, errors.New("LAZ files only support read mode")
	}
	
	lazFile := &LazFile{
		fileName:     fileName,
		fileMode:     fileMode,
		isCompressed: true,
		currentPoint: 0,
	}
	
	// Create LASzip reader
	reader, err := NewLaszipReader()
	if err != nil {
		return nil, fmt.Errorf("failed to create LASzip reader: %v", err)
	}
	lazFile.reader = reader
	
	// Open the LAZ file
	if err := reader.OpenReader(fileName); err != nil {
		return nil, fmt.Errorf("failed to open LAZ file: %v", err)
	}
	
	// Convert LASzip header to LAS header format
	if err := lazFile.convertHeader(); err != nil {
		reader.Close()
		return nil, fmt.Errorf("failed to convert header: %v", err)
	}
	
	return lazFile, nil
}

// convertHeader converts LASzip header to lidario LasHeader format
func (lf *LazFile) convertHeader() error {
	laszipHeader := lf.reader.GetHeader()
	if laszipHeader == nil {
		return errors.New("failed to get LASzip header")
	}
	
	// Convert header fields
	lf.Header = LasHeader{
		FileSignature:        "LASF",
		FileSourceID:         0, // Will need to read from actual header
		GlobalEncoding:       GlobalEncodingField{}, // Will need to read from actual header
		ProjectID1:           0,
		ProjectID2:           0,
		ProjectID3:           0,
		ProjectID4:           [8]byte{},
		VersionMajor:         byte(laszipHeader.VersionMajor),
		VersionMinor:         byte(laszipHeader.VersionMinor),
		SystemID:             "", // 32 characters
		GeneratingSoftware:   "", // 32 characters
		FileCreationDay:      0,
		FileCreationYear:     0,
		HeaderSize:           int(laszipHeader.HeaderSize),
		OffsetToPoints:       int(laszipHeader.OffsetToPointData),
		NumberOfVLRs:         int(laszipHeader.NumberOfVLRs),
		PointFormatID:        byte(laszipHeader.PointDataFormat),
		PointRecordLength:    int(laszipHeader.PointDataRecordLength),
		NumberPoints:         int(laszipHeader.NumberOfPointRecords),
		NumberPointsByReturn: [5]int{}, // Will need to calculate
		XScaleFactor:         laszipHeader.XScaleFactor,
		YScaleFactor:         laszipHeader.YScaleFactor,
		ZScaleFactor:         laszipHeader.ZScaleFactor,
		XOffset:              laszipHeader.XOffset,
		YOffset:              laszipHeader.YOffset,
		ZOffset:              laszipHeader.ZOffset,
		MaxX:                 laszipHeader.MaxX,
		MinX:                 laszipHeader.MinX,
		MaxY:                 laszipHeader.MaxY,
		MinY:                 laszipHeader.MinY,
		MaxZ:                 laszipHeader.MaxZ,
		MinZ:                 laszipHeader.MinZ,
	}
	
	return nil
}

// LasPoint reads a point and converts it to lidario LasPointer format
func (lf *LazFile) LasPoint(pointIndex int) (LasPointer, error) {
	lf.RLock()
	defer lf.RUnlock()
	
	if pointIndex < 0 || pointIndex >= int(lf.Header.NumberPoints) {
		return nil, errors.New("point index out of range")
	}
	
	// For now, we'll read points sequentially
	// TODO: Implement seeking to specific point indices
	if pointIndex != lf.currentPoint {
		return nil, errors.New("random access not yet implemented for LAZ files")
	}
	
	// Read the next point
	if err := lf.reader.ReadPoint(); err != nil {
		return nil, fmt.Errorf("failed to read point: %v", err)
	}
	
	laszipPoint := lf.reader.GetPoint()
	if laszipPoint == nil {
		return nil, errors.New("failed to get point data")
	}
	
	lf.currentPoint++
	
	// Convert LASzip point to lidario format
	return lf.convertPoint(laszipPoint), nil
}

// convertPoint converts LASzip point to lidario LasPointer
func (lf *LazFile) convertPoint(lp *LaszipPoint) LasPointer {
	// Convert coordinates back to real world values
	x := lp.X
	y := lp.Y
	z := lp.Z
	
	// Create bit field from return information
	// Pack the return information into a single byte
	returnByte := lp.ReturnNumber | (lp.NumberOfReturns << 3) | (lp.ScanDirectionFlag << 6) | (lp.EdgeOfFlightFlag << 7)
	bitField := PointBitField{
		Value: returnByte,
	}
	
	// Create classification bit field
	// Pack classification and flags into a single byte
	classificationByte := lp.Classification
	if lp.Classification&0x20 != 0 { // synthetic flag
		classificationByte |= 0x20
	}
	if lp.Classification&0x40 != 0 { // keypoint flag
		classificationByte |= 0x40
	}
	if lp.Classification&0x80 != 0 { // withheld flag
		classificationByte |= 0x80
	}
	classBitField := ClassificationBitField{
		Value: classificationByte,
	}
	
	// Create point record
	pointRecord := &PointRecord0{
		X:             x,
		Y:             y,
		Z:             z,
		Intensity:     lp.Intensity,
		BitField:      bitField,
		ClassBitField: classBitField,
		ScanAngle:     lp.ScanAngleRank,
		UserData:      lp.UserData,
		PointSourceID: lp.PointSourceID,
	}
	
	// Return appropriate point type based on format
	switch lf.Header.PointFormatID {
	case 0:
		return pointRecord
	case 1:
		return &PointRecord1{PointRecord0: pointRecord, GPSTime: lp.GPSTime}
	case 2:
		// TODO: Extract RGB data if available
		rgb := &RgbData{Red: 0, Green: 0, Blue: 0}
		return &PointRecord2{PointRecord0: pointRecord, RGB: rgb}
	case 3:
		// TODO: Extract RGB data if available
		rgb := &RgbData{Red: 0, Green: 0, Blue: 0}
		return &PointRecord3{PointRecord0: pointRecord, GPSTime: lp.GPSTime, RGB: rgb}
	default:
		return pointRecord
	}
}

// GetXYZ gets the coordinates of a specific point
func (lf *LazFile) GetXYZ(pointIndex int) (float64, float64, float64, error) {
	point, err := lf.LasPoint(pointIndex)
	if err != nil {
		return 0, 0, 0, err
	}
	
	pointData := point.PointData()
	return pointData.X, pointData.Y, pointData.Z, nil
}

// Close closes the LAZ file
func (lf *LazFile) Close() error {
	if lf.reader != nil {
		return lf.reader.Close()
	}
	return nil
}

// GetHeader returns the header for LazFile (implement interface)
func (lf *LazFile) GetHeader() *LasHeader {
	return &lf.Header
}

// GetPointCount returns the point count for LazFile (implement interface)
func (lf *LazFile) GetPointCount() uint32 {
	return uint32(lf.Header.NumberPoints)
}

// IsCompressed returns true if this is a compressed LAZ file
func (lf *LazFile) IsCompressed() bool {
	return lf.isCompressed
}

// Ensure LazFile implements LidarFile interface
var _ LidarFile = (*LazFile)(nil)