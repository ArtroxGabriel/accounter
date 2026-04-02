-- internal/platform/migrate/migrations/002_seed_categories.sql

INSERT OR IGNORE INTO categories (name, icon) VALUES
    ('Alimentação',    '🍔'),
    ('Transporte',     '🚗'),
    ('Moradia',        '🏠'),
    ('Saúde',          '💊'),
    ('Educação',       '📚'),
    ('Lazer',          '🎮'),
    ('Vestuário',      '👕'),
    ('Assinaturas',    '📱'),
    ('Outros',         '📦');
