-- +goose Up
SET timezone = 'UTC';

CREATE SCHEMA IF NOT EXISTS budget_schema;
SET search_path TO budget_schema,public;

CREATE TABLE IF NOT EXISTS budget(
	id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY NOT NULL,
	username TEXT NOT NULL,
	shop_name TEXT NOT NULL,
	category TEXT NOT NULL,
	price NUMERIC(9, 2) NOT NULL,
	expense_date DATE NOT NULL
);

CREATE TABLE IF NOT EXISTS salary(
	id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY NOT NULL,
	username TEXT NOT NULL,
	salary NUMERIC(9, 2) NOT NULL,
	store_date DATE NOT NULL
);


-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS salary;
DROP TABLE IF EXISTS budget;
-- +goose StatementEnd