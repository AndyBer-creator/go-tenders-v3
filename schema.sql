CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- cleanup
DROP TABLE IF EXISTS bid_decision CASCADE;
DROP TABLE IF EXISTS bid_feedback CASCADE;
DROP TABLE IF EXISTS bid_version CASCADE;
DROP TABLE IF EXISTS bid CASCADE;
DROP TABLE IF EXISTS tender_version CASCADE;
DROP TABLE IF EXISTS tender CASCADE;
DROP TABLE IF EXISTS organization_responsible CASCADE;
DROP TABLE IF EXISTS organization CASCADE;
DROP TABLE IF EXISTS employee CASCADE;

DROP TYPE IF EXISTS bid_decision_type;
DROP TYPE IF EXISTS bid_status;
DROP TYPE IF EXISTS bid_author_type;
DROP TYPE IF EXISTS tender_status;
DROP TYPE IF EXISTS tender_service_type;
DROP TYPE IF EXISTS organization_type;

-- existing entities from specification
CREATE TABLE employee (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    first_name VARCHAR(50),
    last_name VARCHAR(50),
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE TYPE organization_type AS ENUM (
    'IE',
    'LLC',
    'JSC'
);

CREATE TABLE organization (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    type organization_type,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE organization_responsible (
    id SERIAL PRIMARY KEY,
    organization_id INT NOT NULL REFERENCES organization(id) ON DELETE CASCADE,
    user_id INT NOT NULL REFERENCES employee(id) ON DELETE CASCADE,
    UNIQUE (organization_id, user_id)
);

-- domain enums
CREATE TYPE tender_service_type AS ENUM (
    'Construction',
    'Delivery',
    'Manufacture'
);

CREATE TYPE tender_status AS ENUM (
    'Created',
    'Published',
    'Closed'
);

CREATE TYPE bid_author_type AS ENUM (
    'Organization',
    'User'
);

CREATE TYPE bid_status AS ENUM (
    'Created',
    'Published',
    'Canceled',
    'Approved',
    'Rejected'
);

CREATE TYPE bid_decision_type AS ENUM (
    'Approved',
    'Rejected'
);

-- tenders
CREATE TABLE tender (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    description VARCHAR(500) NOT NULL,
    service_type tender_service_type NOT NULL,
    status tender_status NOT NULL DEFAULT 'Created',
    organization_id INT NOT NULL REFERENCES organization(id) ON DELETE RESTRICT,
    creator_username VARCHAR(50) NOT NULL REFERENCES employee(username) ON DELETE RESTRICT,
    version INT NOT NULL DEFAULT 1 CHECK (version >= 1),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE tender_version (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tender_id UUID NOT NULL REFERENCES tender(id) ON DELETE CASCADE,
    version INT NOT NULL CHECK (version >= 1),
    name VARCHAR(100) NOT NULL,
    description VARCHAR(500) NOT NULL,
    service_type tender_service_type NOT NULL,
    status tender_status NOT NULL,
    organization_id INT NOT NULL,
    creator_username VARCHAR(50) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (tender_id, version)
);

-- bids
CREATE TABLE bid (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    description VARCHAR(500) NOT NULL,
    status bid_status NOT NULL DEFAULT 'Created',
    tender_id UUID NOT NULL REFERENCES tender(id) ON DELETE CASCADE,
    author_type bid_author_type NOT NULL,
    -- For Organization author: organization.id as text, for User author: employee.id as text.
    author_id VARCHAR(100) NOT NULL,
    creator_username VARCHAR(50) NOT NULL REFERENCES employee(username) ON DELETE RESTRICT,
    version INT NOT NULL DEFAULT 1 CHECK (version >= 1),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE bid_version (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bid_id UUID NOT NULL REFERENCES bid(id) ON DELETE CASCADE,
    version INT NOT NULL CHECK (version >= 1),
    name VARCHAR(100) NOT NULL,
    description VARCHAR(500) NOT NULL,
    status bid_status NOT NULL,
    tender_id UUID NOT NULL,
    author_type bid_author_type NOT NULL,
    author_id VARCHAR(100) NOT NULL,
    creator_username VARCHAR(50) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (bid_id, version)
);

CREATE TABLE bid_decision (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bid_id UUID NOT NULL REFERENCES bid(id) ON DELETE CASCADE,
    username VARCHAR(50) NOT NULL REFERENCES employee(username) ON DELETE RESTRICT,
    decision bid_decision_type NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (bid_id, username)
);

CREATE TABLE bid_feedback (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bid_id UUID NOT NULL REFERENCES bid(id) ON DELETE CASCADE,
    description VARCHAR(1000) NOT NULL,
    author_username VARCHAR(50) NOT NULL REFERENCES employee(username) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- indexes
CREATE INDEX idx_tender_org_id ON tender(organization_id);
CREATE INDEX idx_tender_creator ON tender(creator_username);
CREATE INDEX idx_tender_name ON tender(name);

CREATE INDEX idx_bid_tender_id ON bid(tender_id);
CREATE INDEX idx_bid_creator ON bid(creator_username);
CREATE INDEX idx_bid_status ON bid(status);
CREATE INDEX idx_bid_name ON bid(name);

CREATE INDEX idx_bid_decision_bid_id ON bid_decision(bid_id);
CREATE INDEX idx_bid_feedback_bid_id ON bid_feedback(bid_id);
