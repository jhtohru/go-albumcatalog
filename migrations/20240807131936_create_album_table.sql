-- +goose Up
-- +goose StatementBegin
CREATE TABLE album (
	id			uuid PRIMARY KEY,
	title		varchar (255) NOT NULL,
	artist 		varchar (255) NOT NULL,
	price		double precision NOT NULL,
	created_at	timestamp NOT NULL,
	updated_at	timestamp NOT NULL
);

CREATE INDEX album_title_index ON album (title);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX album_title_index;

DROP TABLE album;
-- +goose StatementEnd
