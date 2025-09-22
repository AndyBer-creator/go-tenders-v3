-- Удаляем таблицы, если они есть, чтобы избежать ошибок при повторном выполнении
DROP TABLE IF EXISTS bid_feedback CASCADE;
DROP TABLE IF EXISTS bid_history CASCADE;
DROP TABLE IF EXISTS bid CASCADE;
DROP TABLE IF EXISTS tender_history CASCADE;
DROP TABLE IF EXISTS tender CASCADE;
DROP TABLE IF EXISTS organization_responsible CASCADE;
DROP TABLE IF EXISTS organization CASCADE;
DROP TABLE IF EXISTS employee CASCADE;
DROP TYPE IF EXISTS organization_type;

-- Таблица сотрудников (users)
CREATE TABLE employee (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    first_name VARCHAR(50),
    last_name VARCHAR(50),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Тип организации
CREATE TYPE organization_type AS ENUM (
    'IE',
    'LLC',
    'JSC'
);

-- Таблица организаций
CREATE TABLE organization (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    type organization_type,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Ответственные за организацию
CREATE TABLE organization_responsible (
    id SERIAL PRIMARY KEY,
    organization_id INT REFERENCES organization(id) ON DELETE CASCADE,
    user_id INT REFERENCES employee(id) ON DELETE CASCADE
);

-- Таблица тендеров
CREATE TABLE tender (
    id SERIAL PRIMARY KEY,
    organization_id INT NOT NULL REFERENCES organization(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'CREATED',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Таблица предложений на тендер
CREATE TABLE bid (
    id SERIAL PRIMARY KEY,
    tender_id INT NOT NULL REFERENCES tender(id) ON DELETE CASCADE,
    user_id INT NOT NULL REFERENCES employee(id),
    amount NUMERIC(12, 2),
    description TEXT,
    status VARCHAR(50) DEFAULT 'PENDING',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Отзывы к предложениям
CREATE TABLE bid_feedback (
    id SERIAL PRIMARY KEY,
    bid_id INT NOT NULL REFERENCES bid(id) ON DELETE CASCADE,
    feedback_text TEXT,
    rating INT CHECK (rating BETWEEN 1 AND 5),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- История тендеров
CREATE TABLE tender_history (
    id SERIAL PRIMARY KEY,
    tender_id INT NOT NULL REFERENCES tender(id) ON DELETE CASCADE,
    version INT NOT NULL,
    data JSONB NOT NULL,
    archived_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- История предложений
CREATE TABLE bid_history (
    id SERIAL PRIMARY KEY,
    bid_id INT NOT NULL REFERENCES bid(id) ON DELETE CASCADE,
    version INT NOT NULL,
    data JSONB NOT NULL,
    archived_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Индексы для производительности
CREATE INDEX idx_bid_user_id ON bid(user_id);
CREATE INDEX idx_bid_tender_id ON bid(tender_id);
CREATE INDEX idx_tender_organization_id ON tender(organization_id);
