package runner

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type DataLoader struct {
	filePath string
	fileType string
	headers  []string
}

func NewDataLoader(filePath string) (*DataLoader, error) {
	ext := strings.ToLower(filepath.Ext(filePath))

	var fileType string
	switch ext {
	case ".csv":
		fileType = "csv"
	case ".json":
		fileType = "json"
	default:
		return nil, fmt.Errorf("unsupported data file type: %s (supported: .csv, .json)", ext)
	}

	return &DataLoader{
		filePath: filePath,
		fileType: fileType,
	}, nil
}

func (dl *DataLoader) Headers() []string {
	return dl.headers
}

func (dl *DataLoader) ReadAll() ([]map[string]string, error) {
	switch dl.fileType {
	case "csv":
		return dl.readCSV()
	case "json":
		return dl.readJSON()
	default:
		return nil, fmt.Errorf("unsupported file type: %s", dl.fileType)
	}
}

func (dl *DataLoader) Iterate(fn func(row map[string]string) error) error {
	switch dl.fileType {
	case "csv":
		return dl.iterateCSV(fn)
	case "json":
		return dl.iterateJSON(fn)
	default:
		return fmt.Errorf("unsupported file type: %s", dl.fileType)
	}
}

func (dl *DataLoader) readCSV() ([]map[string]string, error) {
	file, err := os.Open(dl.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(bufio.NewReader(file))

	headers, err := reader.Read()
	if err != nil {
		if err == io.EOF {
			return []map[string]string{}, nil
		}
		return nil, fmt.Errorf("failed to read CSV headers: %w", err)
	}
	dl.headers = headers

	var rows []map[string]string
	rowNum := 1

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read CSV row %d: %w", rowNum, err)
		}

		row := make(map[string]string)
		for i, val := range record {
			if i < len(headers) {
				row[headers[i]] = val
			}
		}
		rows = append(rows, row)
		rowNum++
	}

	return rows, nil
}

func (dl *DataLoader) iterateCSV(fn func(row map[string]string) error) error {
	file, err := os.Open(dl.filePath)
	if err != nil {
		return fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(bufio.NewReader(file))

	headers, err := reader.Read()
	if err != nil {
		if err == io.EOF {
			return nil
		}
		return fmt.Errorf("failed to read CSV headers: %w", err)
	}
	dl.headers = headers

	rowNum := 1
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read CSV row %d: %w", rowNum, err)
		}

		row := make(map[string]string)
		for i, val := range record {
			if i < len(headers) {
				row[headers[i]] = val
			}
		}

		if err := fn(row); err != nil {
			return fmt.Errorf("iterator error at row %d: %w", rowNum, err)
		}
		rowNum++
	}

	return nil
}

func (dl *DataLoader) readJSON() ([]map[string]string, error) {
	file, err := os.Open(dl.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open JSON file: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)

	_, err = decoder.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to read JSON array start: %w", err)
	}

	var rows []map[string]string

	for decoder.More() {
		var obj map[string]interface{}
		if err := decoder.Decode(&obj); err != nil {
			return nil, fmt.Errorf("failed to decode JSON object: %w", err)
		}

		row := make(map[string]string)
		for k, v := range obj {
			row[k] = fmt.Sprintf("%v", v)
		}
		rows = append(rows, row)
	}

	return rows, nil
}

func (dl *DataLoader) iterateJSON(fn func(row map[string]string) error) error {
	file, err := os.Open(dl.filePath)
	if err != nil {
		return fmt.Errorf("failed to open JSON file: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)

	_, err = decoder.Token()
	if err != nil {
		return fmt.Errorf("failed to read JSON array start: %w", err)
	}

	rowNum := 0
	for decoder.More() {
		var obj map[string]interface{}
		if err := decoder.Decode(&obj); err != nil {
			return fmt.Errorf("failed to decode JSON object at row %d: %w", rowNum, err)
		}

		row := make(map[string]string)
		for k, v := range obj {
			row[k] = fmt.Sprintf("%v", v)
		}

		if err := fn(row); err != nil {
			return fmt.Errorf("iterator error at row %d: %w", rowNum, err)
		}
		rowNum++
	}

	return nil
}

type DataConfig struct {
	FilePath string
}
