create table "order" (
    duration integer
    , is_buy_order boolean
    , issued text
    , location_id integer
    , min_volume integer
    , order_id integer
    , price double precision
    , range text
    , system_id integer
    , type_id integer
    , volume_remain integer
    , volume_total integer
);

---- create above / drop below ----

drop table "order";