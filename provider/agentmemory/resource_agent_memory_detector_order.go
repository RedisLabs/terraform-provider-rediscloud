package agentmemory

// Configured detector lists use Terraform order even when the API returns another order.
func orderDetectorSelectionsLikeExisting(models, existing []AgentMemoryDetectorSelectionModel) []AgentMemoryDetectorSelectionModel {
	if len(models) == 0 || len(existing) == 0 {
		return models
	}
	byID := make(map[string]AgentMemoryDetectorSelectionModel, len(models))
	for _, model := range models {
		id, known := knownString(model.ID)
		if !known || id == "" {
			return models
		}
		if _, duplicate := byID[id]; duplicate {
			// Ambiguous identities must remain visible rather than being deduplicated.
			return models
		}
		byID[id] = model
	}
	seen := make(map[string]struct{}, len(existing))
	ordered := make([]AgentMemoryDetectorSelectionModel, 0, len(models))
	for _, previous := range existing {
		id, known := knownString(previous.ID)
		if !known || id == "" {
			return models
		}
		if _, duplicate := seen[id]; duplicate {
			return models
		}
		seen[id] = struct{}{}
		if model, found := byID[id]; found {
			ordered = append(ordered, model)
			delete(byID, id)
		}
	}
	// Preserve new API entries so ordering cannot hide remote drift.
	for _, model := range models {
		if _, remaining := byID[model.ID.ValueString()]; remaining {
			ordered = append(ordered, model)
		}
	}
	return ordered
}

func orderCustomDetectorsLikeExisting(models, existing []AgentMemoryCustomDetectorModel) []AgentMemoryCustomDetectorModel {
	if len(models) == 0 || len(existing) == 0 {
		return models
	}
	byName := make(map[string]AgentMemoryCustomDetectorModel, len(models))
	// An empty name accepted by the API is still a known identity.
	for _, model := range models {
		name, known := knownString(model.Name)
		if !known {
			return models
		}
		if _, duplicate := byName[name]; duplicate {
			// Ambiguous identities must remain visible rather than being deduplicated.
			return models
		}
		byName[name] = model
	}
	seen := make(map[string]struct{}, len(existing))
	ordered := make([]AgentMemoryCustomDetectorModel, 0, len(models))
	for _, previous := range existing {
		name, known := knownString(previous.Name)
		if !known {
			return models
		}
		if _, duplicate := seen[name]; duplicate {
			return models
		}
		seen[name] = struct{}{}
		if model, found := byName[name]; found {
			ordered = append(ordered, model)
			delete(byName, name)
		}
	}
	// Preserve new API entries so ordering cannot hide remote drift.
	for _, model := range models {
		if _, remaining := byName[model.Name.ValueString()]; remaining {
			ordered = append(ordered, model)
		}
	}
	return ordered
}
