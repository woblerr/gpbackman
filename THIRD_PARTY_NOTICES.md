# Third-party Notices

This project includes code and documentation changes adapted from Apache
Cloudberry Backup.

## Apache Cloudberry Backup

Source repository: https://github.com/apache/cloudberry-backup

Source pull request: https://github.com/apache/cloudberry-backup/pull/97

License: Apache License, Version 2.0

Copyright: The Apache Software Foundation

The following parts of this project contain changes adapted from the upstream
Apache Cloudberry Backup gpbackman implementation:

- history database path resolution for `--auto-load-history-db`
- history database opening behavior with `mode=rw`
- user-facing history database error handling
- related command help text, documentation, and tests

The imported changes were modified to fit this project's structure, naming,
message handling, and Greenplum compatibility requirements.

A copy of the Apache License, Version 2.0 is provided in:

`third_party_licenses/APACHE-2.0.txt`
