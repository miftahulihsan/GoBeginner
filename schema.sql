DROP TABLE IF EXISTS items;
DROP TABLE IF EXISTS users;

CREATE TABLE users (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    nik TEXT NOT NULL UNIQUE,
    email TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    division TEXT NOT NULL,
    password TEXT NOT NULL
);

INSERT INTO users (nik, email, name, division, password) VALUES
    ('P95912', 'john.doe@example.com', 'John Doe', 'Engineering', '$2a$10$7zB3cT6wU6b/g7JvGkQ3EO.a4r7.sQ/P9sK6B1z4e2.c/fG8j2W9S'),
    ('P95914', 'bob.johnson@example.com', 'Bob Johnson', 'Sales', '$2a$10$7zB3cT6wU6b/g7JvGkQ3EO.a4r7.sQ/P9sK6B1z4e2.c/fG8j2W9S'),
    ('P95915', 'alice.brown@example.com', 'Alice Brown', 'Human Resources', '$2a$10$7zB3cT6wU6b/g7JvGkQ3EO.a4r7.sQ/P9sK6B1z4e2.c/fG8j2W9S'),
    ('P95913', 'jane.smith@example.com', 'Jane Smith', 'Marketing', '$2a$10$7zB3cT6wU6b/g7JvGkQ3EO.a4r7.sQ/P9sK6B1z4e2.c/fG8j2W9S');

