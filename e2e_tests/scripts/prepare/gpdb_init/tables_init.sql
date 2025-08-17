CREATE SCHEMA IF NOT EXISTS sch1;
CREATE SCHEMA IF NOT EXISTS sch2;

DROP TABLE IF EXISTS sch1.tbl_a;
DROP TABLE IF EXISTS sch1.tbl_b;
DROP TABLE IF EXISTS sch2.tbl_c;
DROP TABLE IF EXISTS sch2.tbl_d;

CREATE TABLE sch1.tbl_a AS SELECT i FROM generate_series(1,100000) AS i;
CREATE TABLE sch1.tbl_b AS SELECT i FROM generate_series(1,100000) AS i;

CREATE TABLE sch2.tbl_c (a int, b int) WITH (appendoptimized = true) DISTRIBUTED BY (a);
INSERT INTO sch2.tbl_c  SELECT i, i FROM generate_series(1,100000) i;

CREATE TABLE sch2.tbl_d (a int, b int) WITH (appendoptimized = true, orientation = column) DISTRIBUTED BY (a);
INSERT INTO sch2.tbl_d SELECT i, i FROM generate_series(1,100000) i;
