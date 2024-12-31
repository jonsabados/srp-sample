create table creatures (
    id bigserial not null primary key,
    name varchar not null,
    description text,
    constraint ux_creatures_name unique(name)
)