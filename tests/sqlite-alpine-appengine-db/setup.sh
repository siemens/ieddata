#!/bin/sh
mkdir -p ${APPENGDBPATH}
sqlite3 ${APPENGDBPATH}/platformbox.db <<END_SQL
CREATE TABLE device ('deviceKey' varchar(32) NOT NULL, 'deviceValue' text NOT NULL, UNIQUE('deviceKey'));
INSERT INTO device VALUES('deviceName', 'canary');
INSERT INTO device VALUES('foo', 'bar');
END_SQL
