-- +goose Up
CREATE TABLE IF NOT EXISTS budget_schema.expense(
	id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY NOT NULL,
	username TEXT NOT NULL,
	shop_name TEXT NOT NULL,
	category TEXT NOT NULL,
	price double precision NOT NULL,
	expense_date DATE NOT NULL
);


-- +goose Down
DROP TABLE IF EXISTS budget CASCADE;