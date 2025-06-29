package lidario

import (
	"testing"
)

func TestLazFileReading(t *testing.T) {
	// Test with the LAZ file in the parent directory
	fileName := "../PNOA_2020_AND_288-4006_ORT-CLA-IRC.laz"
	
	// Test file detection
	if !isLazFile(fileName) {
		t.Fatal("File should be detected as LAZ")
	}
	
	// Open LAZ file using the new interface
	lidarFile, err := NewLidarFile(fileName, "r")
	if err != nil {
		t.Fatalf("Failed to open LAZ file: %v", err)
	}
	defer lidarFile.Close()
	
	// Verify it's compressed
	if !lidarFile.IsCompressed() {
		t.Error("File should be detected as compressed")
	}
	
	// Get header information
	header := lidarFile.GetHeader()
	if header == nil {
		t.Fatal("Failed to get header")
	}
	
	t.Logf("LAZ File Header:")
	t.Logf("Point Format: %d", header.PointFormatID)
	t.Logf("Number of Points: %d", header.NumberPoints)
	t.Logf("X Scale Factor: %f", header.XScaleFactor)
	t.Logf("Y Scale Factor: %f", header.YScaleFactor)
	t.Logf("Z Scale Factor: %f", header.ZScaleFactor)
	
	// Test reading first few points
	pointsToRead := 10
	if header.NumberPoints < pointsToRead {
		pointsToRead = header.NumberPoints
	}
	
	for i := 0; i < pointsToRead; i++ {
		x, y, z, err := lidarFile.GetXYZ(i)
		if err != nil {
			t.Fatalf("Failed to read point %d: %v", i, err)
		}
		
		if i < 3 {
			t.Logf("Point %d: (%.2f, %.2f, %.2f)", i, x, y, z)
		}
		
		// Basic sanity checks
		if x == 0 && y == 0 && z == 0 {
			t.Errorf("Point %d has zero coordinates", i)
		}
	}
}

func TestFileTypeDetection(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"../PNOA_2020_AND_288-4006_ORT-CLA-IRC.laz", "LAZ"},
		{"../PNOA_2020_AND_288-4006_ORT-CLA-IRC.copc.laz", "LAZ"},
		{"testdata/sample.las", "LAS"},
		{"nonexistent.txt", "UNKNOWN"},
	}
	
	for _, test := range tests {
		result := GetFileType(test.filename)
		if result != test.expected {
			t.Errorf("GetFileType(%s) = %s, expected %s", test.filename, result, test.expected)
		}
	}
}