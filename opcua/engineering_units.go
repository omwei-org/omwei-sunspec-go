package opcua

import (
	"fmt"
	"strings"
)

// EngineeringUnitID identifies an OPC UA EUInformation unit by its standard
// namespace URI and numeric UnitID. DisplayName and Description are deliberately
// excluded because they are localized, human-readable labels, not stable keys.
type EngineeringUnitID struct {
	NamespaceURI string
	UnitID       int32
}

// ResolveEngineeringUnit resolves EUInformation metadata through an explicit,
// provisioned allowlist. The mapping's values are the source-unit strings
// understood by the OMWEI source mapping (for example, "W").
//
// This function does not assume a universal numeric UnitID registry, infer a
// unit from DisplayName, or convert between units. The caller must validate
// the namespace URI and UnitID against the chosen OPC UA/UN-CEFACT registry
// and configuration before accepting the returned unit.
func ResolveEngineeringUnit(namespaceURI string, unitID int32, configured map[EngineeringUnitID]string) (string, error) {
	key := EngineeringUnitID{NamespaceURI: strings.TrimSpace(namespaceURI), UnitID: unitID}
	if key.NamespaceURI == "" {
		return "", fmt.Errorf("%w: EUInformation NamespaceURI is empty", ErrMissingUnit)
	}
	unit, ok := configured[key]
	unit = strings.TrimSpace(unit)
	if !ok || unit == "" {
		return "", fmt.Errorf("%w: no configured mapping for namespace %q and UnitID %d", ErrMissingUnit, key.NamespaceURI, unitID)
	}
	return unit, nil
}
