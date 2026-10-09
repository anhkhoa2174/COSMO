package contact

import "time"

// profileReservedKeys are stored elsewhere or managed by the system and may
// not be set through the request's profile map.
var profileReservedKeys = map[string]bool{
	"id": true, "user_id": true, "organization_id": true, "tags": true,
	"do_not_contact": true, "confirmed_facts": true, "ai_insights": true,
	"insight_validation": true, "scores": true, "profile": true,
	"created_at": true, "updated_at": true, "is_deleted": true,
}

// mergeProfileRequest copies the request's nested profile map into the
// profile being stored. Both converters built the profile from the request's
// top-level fields and then deleted the "profile" key, so interests, research
// findings, a company description or custom fields sent under profile were
// silently dropped, and the enrichment prompt had nothing to ground on.
//
// A top-level field already set keeps its value: the standard columns win
// over a stray "email" or "company" inside the map. custom_fields entries are
// merged with any built from extra fields, plain values wrapped into the
// {value, updated_at} form the rest of the system reads.
func mergeProfileRequest(profile map[string]interface{}, extra map[string]interface{}) {
	for key, value := range extra {
		if profileReservedKeys[key] || value == nil {
			continue
		}
		if key == "custom_fields" {
			incoming, ok := value.(map[string]interface{})
			if !ok {
				continue
			}
			existing, _ := profile["custom_fields"].(map[string]interface{})
			if existing == nil {
				existing = make(map[string]interface{}, len(incoming))
			}
			for k, v := range incoming {
				if k == "id" || v == nil {
					continue
				}
				if m, isMap := v.(map[string]interface{}); isMap {
					if _, hasValue := m["value"]; hasValue {
						existing[k] = m
						continue
					}
				}
				existing[k] = map[string]interface{}{
					"value":      v,
					"updated_at": time.Now().UTC().Format(time.RFC3339),
				}
			}
			profile["custom_fields"] = existing
			continue
		}
		if cur, present := profile[key]; present {
			if s, isStr := cur.(string); !isStr || s != "" {
				continue
			}
		}
		profile[key] = value
	}
}
