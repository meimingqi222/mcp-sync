package services

import (
	"encoding/json"
	"fmt"
)

// ConfigConverter handles conversion between different MCP config formats
type ConfigConverter struct {
	configLoader *ConfigLoader
}

// NewConfigConverter creates a new ConfigConverter instance
func NewConfigConverter(configLoader *ConfigLoader) *ConfigConverter {
	return &ConfigConverter{
		configLoader: configLoader,
	}
}

// ConversionResult represents the result of a config conversion
type ConversionResult struct {
	SourceFormat   string                 `json:"source_format"`
	TargetFormat   string                 `json:"target_format"`
	SourceAgent    string                 `json:"source_agent"`
	TargetAgent    string                 `json:"target_agent"`
	OriginalConfig map[string]interface{} `json:"original_config"`
	ConvertedConfig map[string]interface{} `json:"converted_config"`
	Success        bool                   `json:"success"`
	Message        string                 `json:"message"`
}

// ConvertAgentConfig converts MCP config from one agent format to another
func (c *ConfigConverter) ConvertAgentConfig(sourceAgentID, targetAgentID string, sourceConfig map[string]interface{}) (*ConversionResult, error) {
	result := &ConversionResult{
		SourceAgent:     sourceAgentID,
		TargetAgent:     targetAgentID,
		OriginalConfig:  sourceConfig,
		Success:         false,
	}

	// Get agent definitions
	sourceAgent := c.configLoader.GetAgentDefinition(sourceAgentID)
	if sourceAgent == nil {
		err := fmt.Errorf("source agent not found: %s", sourceAgentID)
		result.Message = fmt.Sprintf("Source agent not found: %v", err)
		return result, err
	}

	targetAgent := c.configLoader.GetAgentDefinition(targetAgentID)
	if targetAgent == nil {
		err := fmt.Errorf("target agent not found: %s", targetAgentID)
		result.Message = fmt.Sprintf("Target agent not found: %v", err)
		return result, err
	}

	result.SourceFormat = sourceAgent.Format
	result.TargetFormat = targetAgent.Format

	// If formats are the same, no conversion needed
	if sourceAgent.Format == targetAgent.Format {
		result.ConvertedConfig = sourceConfig
		result.Success = true
		result.Message = "No conversion needed - formats are identical"
		return result, nil
	}

	// Apply format conversion
	transformKey := fmt.Sprintf("%s_to_%s", sourceAgent.Format, targetAgent.Format)
	transform := c.configLoader.GetTransformRule(sourceAgent.Format, targetAgent.Format)

	if transform == nil {
		// Try to convert through standard format as intermediate
		if sourceAgent.Format != "standard" && targetAgent.Format != "standard" {
			// Source -> Standard -> Target
			intermediateResult, err := c.convertToStandard(sourceAgentID, sourceConfig)
			if err != nil {
				result.Message = fmt.Sprintf("Failed intermediate conversion: %v", err)
				return result, err
			}
			return c.convertFromStandard(targetAgentID, intermediateResult)
		}

		result.Message = fmt.Sprintf("No transform rule found: %s", transformKey)
		return result, fmt.Errorf("transform not found: %s", transformKey)
	}

	convertedConfig := c.applyTransform(sourceConfig, transform)
	result.ConvertedConfig = convertedConfig
	result.Success = true
	result.Message = fmt.Sprintf("Successfully converted from %s to %s format", sourceAgent.Format, targetAgent.Format)

	return result, nil
}

// ConvertToCodex converts any agent config to Codex format
func (c *ConfigConverter) ConvertToCodex(sourceAgentID string, sourceConfig map[string]interface{}) (*ConversionResult, error) {
	return c.ConvertAgentConfig(sourceAgentID, "codex", sourceConfig)
}

// ConvertFromCodex converts Codex config to any agent format
func (c *ConfigConverter) ConvertFromCodex(targetAgentID string, codexConfig map[string]interface{}) (*ConversionResult, error) {
	return c.ConvertAgentConfig("codex", targetAgentID, codexConfig)
}

// convertToStandard converts any format to standard format
func (c *ConfigConverter) convertToStandard(sourceAgentID string, sourceConfig map[string]interface{}) (map[string]interface{}, error) {
	sourceAgent := c.configLoader.GetAgentDefinition(sourceAgentID)
	if sourceAgent == nil {
		return nil, fmt.Errorf("source agent not found: %s", sourceAgentID)
	}

	if sourceAgent.Format == "standard" {
		return sourceConfig, nil
	}

	transform := c.configLoader.GetTransformRule(sourceAgent.Format, "standard")
	
	if transform == nil {
		return nil, fmt.Errorf("no transform to standard from %s", sourceAgent.Format)
	}

	return c.applyTransform(sourceConfig, transform), nil
}

// convertFromStandard converts standard format to any format
func (c *ConfigConverter) convertFromStandard(targetAgentID string, standardConfig map[string]interface{}) (*ConversionResult, error) {
	targetAgent := c.configLoader.GetAgentDefinition(targetAgentID)
	if targetAgent == nil {
		return nil, fmt.Errorf("target agent not found: %s", targetAgentID)
	}

	result := &ConversionResult{
		SourceAgent:     "standard",
		TargetAgent:     targetAgentID,
		SourceFormat:    "standard",
		TargetFormat:    targetAgent.Format,
		OriginalConfig:  standardConfig,
	}

	if targetAgent.Format == "standard" {
		result.ConvertedConfig = standardConfig
		result.Success = true
		result.Message = "No conversion needed"
		return result, nil
	}

	transform := c.configLoader.GetTransformRule("standard", targetAgent.Format)
	
	if transform == nil {
		result.Message = fmt.Sprintf("No transform from standard to %s", targetAgent.Format)
		return result, fmt.Errorf("no transform from standard to %s", targetAgent.Format)
	}

	result.ConvertedConfig = c.applyTransform(standardConfig, transform)
	result.Success = true
	result.Message = fmt.Sprintf("Successfully converted to %s format", targetAgent.Format)

	return result, nil
}

// applyTransform applies transformation rules to config
func (c *ConfigConverter) applyTransform(config map[string]interface{}, transform *TransformRule) map[string]interface{} {
	result := make(map[string]interface{})

	// Process each server in config
	for serverName, serverConfigInterface := range config {
		serverConfig, ok := serverConfigInterface.(map[string]interface{})
		if !ok {
			result[serverName] = serverConfigInterface
			continue
		}

		transformedServer := make(map[string]interface{})

		// Apply keep_fields if specified
		if len(transform.KeepFields) > 0 {
			for _, field := range transform.KeepFields {
				if val, exists := serverConfig[field]; exists {
					transformedServer[field] = val
				}
			}
		} else {
			// Keep all fields if keep_fields not specified
			for key, val := range serverConfig {
				transformedServer[key] = val
			}
		}

		// Remove fields specified in remove_fields
		for _, field := range transform.RemoveFields {
			delete(transformedServer, field)
		}

		// Add new fields
		for key, val := range transform.AddFields {
			transformedServer[key] = val
		}

		result[serverName] = transformedServer
	}

	return result
}

// BatchConvertConfig converts config to multiple target formats
func (c *ConfigConverter) BatchConvertConfig(sourceAgentID string, sourceConfig map[string]interface{}, targetAgentIDs []string) ([]*ConversionResult, error) {
	results := make([]*ConversionResult, 0, len(targetAgentIDs))

	for _, targetID := range targetAgentIDs {
		result, err := c.ConvertAgentConfig(sourceAgentID, targetID, sourceConfig)
		if err != nil {
			// Continue with other conversions even if one fails
			result.Success = false
			result.Message = fmt.Sprintf("Conversion failed: %v", err)
		}
		results = append(results, result)
	}

	return results, nil
}

// ExportConversionAsJSON exports a conversion result as JSON string
func (c *ConfigConverter) ExportConversionAsJSON(result *ConversionResult) (string, error) {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %v", err)
	}
	return string(data), nil
}

// ValidateConfigFormat validates if a config matches expected format
func (c *ConfigConverter) ValidateConfigFormat(agentID string, config map[string]interface{}) (bool, []string) {
	agent := c.configLoader.GetAgentDefinition(agentID)
	if agent == nil {
		return false, []string{fmt.Sprintf("Agent not found: %s", agentID)}
	}

	errors := []string{}

	// Basic validation for standard format
	if agent.Format == "standard" {
		for serverName, serverConfigInterface := range config {
			serverConfig, ok := serverConfigInterface.(map[string]interface{})
			if !ok {
				errors = append(errors, fmt.Sprintf("Server %s: invalid config structure", serverName))
				continue
			}

			// Check required fields
			if _, hasCommand := serverConfig["command"]; !hasCommand {
				errors = append(errors, fmt.Sprintf("Server %s: missing 'command' field", serverName))
			}
		}
	}

	// Zed format specific validation
	if agent.Format == "zed" {
		for serverName, serverConfigInterface := range config {
			serverConfig, ok := serverConfigInterface.(map[string]interface{})
			if !ok {
				errors = append(errors, fmt.Sprintf("Server %s: invalid config structure", serverName))
				continue
			}

			// Check Zed-specific fields
			if _, hasCommand := serverConfig["command"]; !hasCommand {
				errors = append(errors, fmt.Sprintf("Server %s: missing 'command' field", serverName))
			}
			if _, hasSource := serverConfig["source"]; !hasSource {
				errors = append(errors, fmt.Sprintf("Server %s: missing 'source' field", serverName))
			}
		}
	}

	return len(errors) == 0, errors
}
