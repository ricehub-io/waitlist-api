CREATE TABLE settings (
    id INT PRIMARY KEY DEFAULT 1,
    slots_total INT NOT NULL DEFAULT 10 CHECK (slots_total > 0),
    slots_taken INT NOT NULL DEFAULT 0 CHECK (slots_taken >= 0),
    CONSTRAINT single_row CHECK (id = 1)
);

INSERT INTO settings (slots_total, slots_taken) VALUES (10, 0);