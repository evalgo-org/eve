package tracing

import (
	"encoding/json"
)

// extractMetadata extracts queryable metadata based on action + object type
func (t *Tracer) extractMetadata(actionType, objectType string, requestBody, responseBody []byte) json.RawMessage {
	// Declare metadata map
	var metadata map[string]interface{}

	// Extract based on action + object combination
	switch {
	case actionType == "CreateAction" && objectType == "SoftwareApplication":
		// Container creation
		metadata = extractContainerMetadata(requestBody, responseBody)

	case actionType == "TransferAction" && objectType == "Database":
		// Database migration
		metadata = extractMigrationMetadata(requestBody, responseBody)

	case actionType == "UploadAction" && objectType == "Dataset":
		// Backup operation
		metadata = extractBackupMetadata(requestBody, responseBody)

	case actionType == "ExecuteAction" && objectType == "SoftwareSourceCode":
		// CI/CD build
		metadata = extractBuildMetadata(requestBody, responseBody)

	case actionType == "ReplaceAction" && objectType == "DataFeed":
		// ETL transformation
		metadata = extractETLMetadata(requestBody, responseBody)

	default:
		// Generic metadata extraction
		metadata = extractGenericMetadata(requestBody, responseBody)
	}

	// Convert to JSON
	metadataJSON, _ := json.Marshal(metadata)
	return metadataJSON
}

// extractContainerMetadata extracts metadata for container operations
func extractContainerMetadata(reqBody, respBody []byte) map[string]interface{} {
	var req struct {
		Object struct {
			Image string `json:"image"`
			Name  string `json:"name"`
		} `json:"object"`
	}

	var resp struct {
		Result struct {
			ContainerID string   `json:"containerId"`
			Started     bool     `json:"started"`
			Ports       []string `json:"ports"`
			Health      string   `json:"healthStatus"`
		} `json:"result"`
	}

	json.Unmarshal(reqBody, &req)
	json.Unmarshal(respBody, &resp)

	return map[string]interface{}{
		"container_id":  resp.Result.ContainerID,
		"image":         req.Object.Image,
		"started":       resp.Result.Started,
		"ports":         resp.Result.Ports,
		"health_status": resp.Result.Health,
	}
}

// extractMigrationMetadata extracts metadata for database migrations
func extractMigrationMetadata(reqBody, respBody []byte) map[string]interface{} {
	var req struct {
		Object struct {
			Name string `json:"name"`
		} `json:"object"`
		Target struct {
			Name string `json:"name"`
		} `json:"target"`
	}

	var resp struct {
		Result struct {
			TotalTables     int     `json:"totalTables"`
			CompletedTables int     `json:"completedTables"`
			TotalRows       int     `json:"totalRows"`
			TransferredRows int     `json:"transferredRows"`
			Progress        float64 `json:"progressPercent"`
			CurrentTable    string  `json:"currentTable"`
		} `json:"result"`
	}

	json.Unmarshal(reqBody, &req)
	json.Unmarshal(respBody, &resp)

	return map[string]interface{}{
		"source_database":  req.Object.Name,
		"target_database":  req.Target.Name,
		"total_tables":     resp.Result.TotalTables,
		"completed_tables": resp.Result.CompletedTables,
		"total_rows":       resp.Result.TotalRows,
		"transferred_rows": resp.Result.TransferredRows,
		"progress_percent": resp.Result.Progress,
		"current_table":    resp.Result.CurrentTable,
	}
}

// extractBackupMetadata extracts metadata for backup operations
func extractBackupMetadata(reqBody, respBody []byte) map[string]interface{} {
	var req struct {
		Object struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"object"`
	}

	var resp struct {
		Result struct {
			BackupType string `json:"backupType"`
			SizeBytes  int64  `json:"sizeBytes"`
			Checksum   string `json:"checksum"`
			Location   string `json:"storageLocation"`
			ExpiresAt  string `json:"expiresAt"`
		} `json:"result"`
	}

	json.Unmarshal(reqBody, &req)
	json.Unmarshal(respBody, &resp)

	return map[string]interface{}{
		"backup_type":       resp.Result.BackupType,
		"source_database":   req.Object.Name,
		"backup_size_bytes": resp.Result.SizeBytes,
		"checksum":          resp.Result.Checksum,
		"storage_location":  resp.Result.Location,
		"expires_at":        resp.Result.ExpiresAt,
	}
}

// extractBuildMetadata extracts metadata for CI/CD builds
func extractBuildMetadata(reqBody, respBody []byte) map[string]interface{} {
	var req struct {
		Object struct {
			CodeRepository string `json:"codeRepository"`
		} `json:"object"`
		Instrument struct {
			Branch string `json:"branch"`
			Commit string `json:"commit"`
		} `json:"instrument"`
	}

	var resp struct {
		Result struct {
			BuildNumber    int    `json:"buildNumber"`
			TestsPassed    int    `json:"testsPassed"`
			TestsFailed    int    `json:"testsFailed"`
			ArtifactsCount int    `json:"artifactsCount"`
			CommitSHA      string `json:"commitSha"`
		} `json:"result"`
	}

	json.Unmarshal(reqBody, &req)
	json.Unmarshal(respBody, &resp)

	return map[string]interface{}{
		"repository":      req.Object.CodeRepository,
		"branch":          req.Instrument.Branch,
		"commit_sha":      resp.Result.CommitSHA,
		"build_number":    resp.Result.BuildNumber,
		"tests_passed":    resp.Result.TestsPassed,
		"tests_failed":    resp.Result.TestsFailed,
		"artifacts_count": resp.Result.ArtifactsCount,
	}
}

// extractETLMetadata extracts metadata for ETL transformations
func extractETLMetadata(reqBody, respBody []byte) map[string]interface{} {
	var req struct {
		Object struct {
			Name string `json:"name"`
		} `json:"object"`
	}

	var resp struct {
		Result struct {
			InputRows     int     `json:"inputRows"`
			OutputRows    int     `json:"outputRows"`
			FilteredRows  int     `json:"filteredRows"`
			RowsPerSecond float64 `json:"rowsPerSecond"`
			QualityPassed bool    `json:"dataQualityPassed"`
		} `json:"result"`
	}

	json.Unmarshal(reqBody, &req)
	json.Unmarshal(respBody, &resp)

	return map[string]interface{}{
		"destination_table":   req.Object.Name,
		"input_rows":          resp.Result.InputRows,
		"output_rows":         resp.Result.OutputRows,
		"filtered_rows":       resp.Result.FilteredRows,
		"rows_per_second":     resp.Result.RowsPerSecond,
		"data_quality_passed": resp.Result.QualityPassed,
	}
}

// extractGenericMetadata extracts basic metadata for unknown action types
func extractGenericMetadata(reqBody, respBody []byte) map[string]interface{} {
	var req struct {
		Object struct {
			Name       string `json:"name"`
			Identifier string `json:"identifier"`
		} `json:"object"`
	}

	json.Unmarshal(reqBody, &req)

	return map[string]interface{}{
		"object_name":       req.Object.Name,
		"object_identifier": req.Object.Identifier,
	}
}
