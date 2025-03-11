
CREATE SCHEMA IF NOT EXISTS swift_catalog.default_schema
WITH (location = 'file:///warehouse');


CREATE TABLE IF NOT EXISTS swift_catalog.default_schema.swift_banks (
    swift_code VARCHAR,
    hq_swift_base VARCHAR,
    country_iso_code VARCHAR,
    bank_name VARCHAR,
    entity_type VARCHAR,
    created_at TIMESTAMP,
    updated_at TIMESTAMP
)
WITH (
    partitioning = ARRAY['country_iso_code']
);

-- Create the views using the Iceberg table
CREATE OR REPLACE VIEW swift_catalog.default_schema.v_swift_bank_headquarters AS
SELECT
    swift_code,
    hq_swift_base,
    country_iso_code,
    bank_name,
    created_at,
    updated_at
FROM
    swift_catalog.default_schema.swift_banks
WHERE
    entity_type = 'HEADQUARTERS';

CREATE OR REPLACE VIEW swift_catalog.default_schema.v_swift_bank_branches AS
SELECT
    swift_code,
    hq_swift_base,
    country_iso_code,
    bank_name,
    created_at,
    updated_at
FROM
    swift_catalog.default_schema.swift_banks
WHERE
    entity_type = 'BRANCH';

CREATE OR REPLACE VIEW swift_catalog.default_schema.v_bank_branch_counts AS
SELECT
    h.swift_code AS hq_swift_code,
    h.bank_name,
    h.country_iso_code,
    COUNT(b.swift_code) AS branch_count
FROM
    swift_catalog.default_schema.swift_banks h
    LEFT JOIN swift_catalog.default_schema.swift_banks b
    ON h.hq_swift_base = b.hq_swift_base
    AND b.entity_type = 'BRANCH'
WHERE
    h.entity_type = 'HEADQUARTERS'
GROUP BY
    h.swift_code,
    h.bank_name,
    h.country_iso_code;

-- Add comments for documentation
COMMENT ON TABLE swift_catalog.default_schema.swift_banks
IS 'All bank entities with SWIFT codes, including both headquarters and branches';

COMMENT ON VIEW swift_catalog.default_schema.v_swift_bank_headquarters
IS 'Bank headquarters with SWIFT codes ending in XXX';

COMMENT ON VIEW swift_catalog.default_schema.v_swift_bank_branches
IS 'Bank branches with specific branch codes (not ending in XXX)';
