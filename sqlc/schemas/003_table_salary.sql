-- +goose Up
CREATE TABLE IF NOT EXISTS budget_schema.salary(
	id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY NOT NULL,
	username TEXT NOT NULL,
	salary double precision NOT NULL,
	store_date DATE NOT NULL
);


-- +goose Down
DROP TABLE IF EXISTS salary CASCADE;