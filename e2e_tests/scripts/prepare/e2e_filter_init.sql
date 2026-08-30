BEGIN;

CREATE TABLE IF NOT EXISTS public.e2e_data (
    id integer,
    value text
) DISTRIBUTED BY (id);

TRUNCATE TABLE public.e2e_data;

INSERT INTO public.e2e_data (id, value)
SELECT i, 'e2e-value-' || i
FROM generate_series(1, 100) AS i;

COMMIT;
