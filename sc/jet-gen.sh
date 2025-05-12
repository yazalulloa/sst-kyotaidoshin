#!/bin/bash

DIR=$(dirname "$(readlink -f "$0")")

#!/bin/bash

# Path to the SQLite database file
db_file="$DIR/temp.db"
# Path to the SQL schema file
schema_file="$DIR/../schema/schema.sql"

# Execute the schema from the file
sqlite3 "$db_file" < "$schema_file"

echo "Database created successfully!"

SOURCE_PATH="$DIR"/../packages/backend/kyo-repo/internal/db/gen
jet -source=sqlite -dsn="file:///$db_file" -path="$SOURCE_PATH"
rm "$db_file"