## ADDED Requirements

### Requirement: ECS Client IP Extraction
The DNS request pipeline MUST parse the EDNS Client Subnet option and derive a single effective client IP for routing decisions.

#### Scenario: ECS IPv4 is present and valid
- **WHEN** a DNS query contains a valid ECS IPv4 subnet option
- **THEN** the system SHALL use the ECS client IP as routing input

#### Scenario: ECS is missing
- **WHEN** a DNS query does not include ECS option
- **THEN** the system SHALL fall back to existing source-IP routing behavior without error

#### Scenario: ECS is malformed
- **WHEN** a DNS query includes an invalid ECS option (bad family, invalid prefix, or parse failure)
- **THEN** the system SHALL ignore ECS and continue with fallback routing behavior

### Requirement: Client IP to City Mapping
The routing service MUST map the effective client IP to GoEdge regional metadata and resolve a city identity when mapping data exists.

#### Scenario: IP maps to a known city
- **WHEN** the effective client IP belongs to an IP range mapped to a known city
- **THEN** the system SHALL attach city identity (e.g., cityId/cityName) to routing context

#### Scenario: IP does not map to city
- **WHEN** the effective client IP has no city-level mapping in current IP library
- **THEN** the system SHALL continue routing with non-city strategy and SHALL NOT fail the DNS response

### Requirement: City-First Node Selection with Fallback
The node selection logic MUST prioritize healthy edge nodes in the mapped city and MUST apply deterministic fallback when city candidates are unavailable.

#### Scenario: City has healthy edge nodes
- **WHEN** routing context includes a mapped city with one or more healthy nodes
- **THEN** the DNS answer SHALL be selected from that city candidate set

#### Scenario: City has no healthy edge nodes
- **WHEN** routing context includes a mapped city but no healthy city nodes are available
- **THEN** the system SHALL apply configured fallback policy and return a valid answer from broader candidates

### Requirement: ECS Routing Observability
The system MUST expose observable outcomes for ECS extraction, city mapping, and fallback execution.

#### Scenario: Successful ECS city route
- **WHEN** a request is routed using ECS-derived city context
- **THEN** the system SHALL record a success event/metric for ECS city hit

#### Scenario: Fallback route due to ECS or city miss
- **WHEN** routing falls back because ECS is unavailable/invalid or city candidates are empty
- **THEN** the system SHALL record fallback reason for diagnostics
