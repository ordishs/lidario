package lidario

/*
#cgo CFLAGS: -I/opt/homebrew/opt/laszip/include
#cgo LDFLAGS: -L/opt/homebrew/opt/laszip/lib -llaszip

#include <laszip/laszip_api.h>
#include <stdlib.h>
*/
import "C"

import (
	"errors"
	"unsafe"
)

// LaszipReader wraps the LASzip C API for reading compressed LAZ files
type LaszipReader struct {
	pointer      C.laszip_POINTER
	header       *C.laszip_header_struct
	point        *C.laszip_point_struct
	isOpen       bool
	pointCount   uint64
	currentPoint uint64
}

// NewLaszipReader creates a new LASzip reader
func NewLaszipReader() (*LaszipReader, error) {
	reader := &LaszipReader{}

	// Create LASzip pointer
	if err := reader.create(); err != nil {
		return nil, err
	}

	return reader, nil
}

// create initializes the LASzip pointer
func (r *LaszipReader) create() error {
	result := C.laszip_create(&r.pointer)
	if result != 0 {
		// Get the specific error message
		var errorStr *C.char
		if r.pointer != nil {
			C.laszip_get_error(r.pointer, &errorStr)
			if errorStr != nil {
				errorMsg := C.GoString(errorStr)
				return errors.New("failed to create LASzip pointer: " + errorMsg)
			}
		}
		return errors.New("failed to create LASzip pointer: unknown error")
	}
	return nil
}

// OpenReader opens a LAZ file for reading
func (r *LaszipReader) OpenReader(filename string) error {
	if r.isOpen {
		return errors.New("reader already open")
	}

	cFilename := C.CString(filename)
	defer C.free(unsafe.Pointer(cFilename))

	// Open the reader
	var isCompressed C.laszip_BOOL
	result := C.laszip_open_reader(r.pointer, cFilename, &isCompressed)
	if result != 0 {
		return r.getError()
	}

	// Get header
	result = C.laszip_get_header_pointer(r.pointer, &r.header)
	if result != 0 {
		return r.getError()
	}

	// Get point
	result = C.laszip_get_point_pointer(r.pointer, &r.point)
	if result != 0 {
		return r.getError()
	}

	r.pointCount = uint64(r.header.number_of_point_records)
	r.currentPoint = 0
	r.isOpen = true

	return nil
}

// ReadPoint reads the next point from the LAZ file
func (r *LaszipReader) ReadPoint() error {
	if !r.isOpen {
		return errors.New("reader not open")
	}

	if r.currentPoint >= r.pointCount {
		return errors.New("EOF: no more points")
	}

	result := C.laszip_read_point(r.pointer)
	if result != 0 {
		return r.getError()
	}

	r.currentPoint++
	return nil
}

// GetPoint returns the current point data
func (r *LaszipReader) GetPoint() *LaszipPoint {
	if !r.isOpen || r.point == nil {
		return nil
	}

	// Get the real coordinates using scale and offset
	var coordinates [3]C.laszip_F64
	C.laszip_get_coordinates(r.pointer, &coordinates[0])

	// For now, create a simple point with basic information
	// TODO: Properly extract bit fields from the C structure
	return &LaszipPoint{
		X:                 float64(coordinates[0]),
		Y:                 float64(coordinates[1]),
		Z:                 float64(coordinates[2]),
		Intensity:         uint16(r.point.intensity),
		ReturnNumber:      1, // Default values for now
		NumberOfReturns:   1,
		ScanDirectionFlag: 0,
		EdgeOfFlightFlag:  0,
		Classification:    0,
		ScanAngleRank:     int8(r.point.scan_angle_rank),
		UserData:          uint8(r.point.user_data),
		PointSourceID:     uint16(r.point.point_source_ID),
		GPSTime:           float64(r.point.gps_time),
	}
}

// GetHeader returns the LAZ file header information
func (r *LaszipReader) GetHeader() *LaszipHeader {
	if !r.isOpen || r.header == nil {
		return nil
	}

	return &LaszipHeader{
		VersionMajor:          uint8(r.header.version_major),
		VersionMinor:          uint8(r.header.version_minor),
		HeaderSize:            uint16(r.header.header_size),
		OffsetToPointData:     uint32(r.header.offset_to_point_data),
		NumberOfVLRs:          uint32(r.header.number_of_variable_length_records),
		PointDataFormat:       uint8(r.header.point_data_format),
		PointDataRecordLength: uint16(r.header.point_data_record_length),
		NumberOfPointRecords:  uint32(r.header.number_of_point_records),
		XScaleFactor:          float64(r.header.x_scale_factor),
		YScaleFactor:          float64(r.header.y_scale_factor),
		ZScaleFactor:          float64(r.header.z_scale_factor),
		XOffset:               float64(r.header.x_offset),
		YOffset:               float64(r.header.y_offset),
		ZOffset:               float64(r.header.z_offset),
		MaxX:                  float64(r.header.max_x),
		MinX:                  float64(r.header.min_x),
		MaxY:                  float64(r.header.max_y),
		MinY:                  float64(r.header.min_y),
		MaxZ:                  float64(r.header.max_z),
		MinZ:                  float64(r.header.min_z),
	}
}

// Close closes the LAZ reader
func (r *LaszipReader) Close() error {
	if !r.isOpen {
		return nil
	}

	result := C.laszip_close_reader(r.pointer)
	if result != 0 {
		return r.getError()
	}

	result = C.laszip_destroy(r.pointer)
	if result != 0 {
		return r.getError()
	}

	r.isOpen = false
	return nil
}

// getError retrieves the last error from LASzip
func (r *LaszipReader) getError() error {
	var cError *C.char
	C.laszip_get_error(r.pointer, &cError)
	if cError != nil {
		return errors.New(C.GoString(cError))
	}
	return errors.New("unknown LASzip error")
}

// LaszipPoint represents a point read from a LAZ file
type LaszipPoint struct {
	X                 float64
	Y                 float64
	Z                 float64
	Intensity         uint16
	ReturnNumber      uint8
	NumberOfReturns   uint8
	ScanDirectionFlag uint8
	EdgeOfFlightFlag  uint8
	Classification    uint8
	ScanAngleRank     int8
	UserData          uint8
	PointSourceID     uint16
	GPSTime           float64
}

// LaszipHeader represents the header of a LAZ file
type LaszipHeader struct {
	VersionMajor          uint8
	VersionMinor          uint8
	HeaderSize            uint16
	OffsetToPointData     uint32
	NumberOfVLRs          uint32
	PointDataFormat       uint8
	PointDataRecordLength uint16
	NumberOfPointRecords  uint32
	XScaleFactor          float64
	YScaleFactor          float64
	ZScaleFactor          float64
	XOffset               float64
	YOffset               float64
	ZOffset               float64
	MaxX                  float64
	MinX                  float64
	MaxY                  float64
	MinY                  float64
	MaxZ                  float64
	MinZ                  float64
}
