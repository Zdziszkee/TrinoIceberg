CREATE SCHEMA IF NOT EXISTS swift_catalog.default_schema
WITH (location = 'file:///warehouse');


CREATE TABLE IF NOT EXISTS swift_catalog.default_schema.swift_banks (
    swift_code VARCHAR,
    swift_code_base VARCHAR,
    country_iso_code VARCHAR,
    bank_name VARCHAR,
    is_headquarter BOOLEAN,
    address VARCHAR,
    country_name VARCHAR,
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
    swift_code_base,
    country_iso_code,
    bank_name,
    address,
    country_name,
    created_at,
    updated_at
FROM
    swift_catalog.default_schema.swift_banks
WHERE
    is_headquarter = TRUE;

CREATE OR REPLACE VIEW swift_catalog.default_schema.v_swift_bank_branches AS
SELECT
    swift_code,
    swift_code_base,
    country_iso_code,
    bank_name,
    address,
    country_name,
    created_at,
    updated_at
FROM
    swift_catalog.default_schema.swift_banks
WHERE
    is_headquarter = FALSE;

CREATE OR REPLACE VIEW swift_catalog.default_schema.v_bank_branch_counts AS
SELECT
    h.swift_code AS headquarter_swift_code,
    h.bank_name,
    h.country_iso_code,
    COUNT(b.swift_code) AS branch_count
FROM
    swift_catalog.default_schema.swift_banks h
    LEFT JOIN swift_catalog.default_schema.swift_banks b
    ON h.swift_code_base = b.swift_code_base
    AND b.is_headquarter = FALSE
WHERE
    h.is_headquarter = TRUE
GROUP BY
    h.swift_code,
    h.bank_name,
    h.country_iso_code;

-- Add comments for documentation
COMMENT ON TABLE swift_catalog.default_schema.swift_banks
IS 'All bank entities with SWIFT codes, including both headquarter and branch details';

COMMENT ON VIEW swift_catalog.default_schema.v_swift_bank_headquarters
IS 'Bank headquarters with is_headquarter flag set to true';

COMMENT ON VIEW swift_catalog.default_schema.v_swift_bank_branches
IS 'Bank branches with is_headquarter flag set to false';
