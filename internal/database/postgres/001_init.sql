create table "order" (
    duration bigint
    , is_buy_order boolean
    , issued text
    , location_id bigint
    , min_volume bigint
    , order_id bigint
    , price double precision
    , range text
    , system_id bigint
    , type_id bigint
    , volume_remain bigint
    , volume_total bigint
);

---- create above / drop below ----

drop table "order";