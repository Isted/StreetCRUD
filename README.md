# StreetCRUD

#### Videos:
* A three minute [overview](http://youtu.be/R_FQdpizIDQ "StreetCRUD Demo") that demonstrates StreetCRUD.
* A longer, more detailed [video](http://youtu.be/t3CCy1zSGNw "StreetCRUD Tutorial").

#### Introduction

StreetCRUD is a code and table generation command-line utility for people who aren't fans of ORMs, but appreciate a kick start creating struct methods, tables, and queries for CRUD functionality. You only have to supply your structs, database connection info, and a few keywords in a text file that StreetCRUD will process. Tables, queries, and struct methods will be created or altered automatically. This allows the programmer to add methods and queries at a later date without having to wrestle with an ORM. You keep all the power, but don't have to start at level one. At this time, StreetCRUD supports PostgreSQL, JSON, and Go.

As a nice side benefit StreetCRUD can be used to easily reorder columns in Postgres tables. The documentation below may look daunting, but it's just thorough. StreetCRUD is actually quite simple and straightforward to use.

**Features Include**:
* Table creation (if it doesn't exist), table alteration (if it exists)
* Generates Get, Insert, Update, Patch (optional), GetByIndex (optional), and Delete struct methods (with corresponding queries)
* StreetCRUD can be rerun to alter methods and queries if there is a struct change
* Table data is safely copied via a map if a struct/table is altered
* Methods return and receive JSON
* Reordering of table columns when a struct is altered (something pgAdmin doesn't support)
* Queries can be prepared for optimal performance
* Null columns are supported for most data types
* Great for new gophers learning how Go uses databases. Also useful for understanding data types, structs, and struct methods

**Installation (OS X)**:

~~~
$ go get github.com/isted/StreetCRUD
~~~

If GOBIN isn't set, you can run the following before you install StreetCRUD:
~~~
$ export GOBIN=/Users/userName/go/bin
~~~

Install:
~~~
$ go install github.com/isted/StreetCRUD
~~~
Run:
~~~
$ cd ~/go/bin/StreetCRUD && ./StreetCRUD
~~~

## Getting Started

The user will need to create a text file that defines information about his or her database connection, structs (data models), and keywords. This file will be processed by StreetCRUD and used for generating go code, SQL queries, tables, and columns. Below is an example file that would add two new structs/tables and alter a third struct/table. Inconsistent spacing and capitalization are used on purpose to demonstrate that they aren't an issue for file processing. Keywords are defined below the example file.

~~~
StreetCRUD
[Server] localhost
[User]   dan
[Group] sqlgroupname
[Password]   secret
[Database] db_name
[schema] public
[ssl] false
[Underscore] true
[package]models

[Add struct]
[table]
[File name] XUsers
[prepared] true
type User struct {
	LoginID  int    `json:"loginid"` [primary]
	Name     string `json:"name, omitempty"` [index] [patch] [size:255]
	Email    string `json:"email"`
	Password string `json:"password" out:"false"`
	Deleted bool `json:”deleted”` [deleted]
	DelOn	   time.Time   `json:”deletedon”` [deletedOn][nulls]
}

[add struct]
[table] blog
[File name]
[prepared] false
type Blog struct {
	BlogID     int    `json:”blogid"` [primary]
	Title      string `json:”title, omitempty"` [index][patch][size:255]
	Body       string `json:”body”` [nulls]
	CategoryID int    `json:”catID”`
	Object     T	  [Ignore]
	Deleted    bool   `json:”deleted”` [deleted]
	DeletedOn	   time.Time   `json:”deletedon”` [deletedOn][nulls]

}

[alter table] user
[copy cols]
login_id [to] LogID
name [to] UserName
email [to] Email
[add Struct]
[table] tbl_new_name
[File name] x9.go
[prepared] false
type UserNew struct {
	LogID  int `json:"loginid"`	 	[primary]
	UserName string `json:”userName”` [index][patch] [size:255]
	Phone string `json:”phone”` [nulls]
	Email string `json:”email”` [nulls]
	Password string `json:"password" out:"false"`
}
~~~

#### Keywords and Definitions:

The Following keywords need to appear at the top of the user created text file before structs are defined (one key-value pair per line). A value needs to be typed next to the keyword. For instance, [Server] localhost indicates that the name of the database server is localhost. The keywords defined below will let StreetCRUD connect and make changes to your database server. Some of the other key-values pairs will indicate package and table naming options. Keywords and their values are not case sensitive, also spaces after the "]" symbol are ignored. StreetCRUD includes an example struct files for you to test out and process. You will need to create the database, database user, and enter all relevant server information at the top of the example struct files so that they can be processed properly.

- **[Server]**: Name of the database server where tables should be created (for many: localhost)
- **[User]**: User name used to login to the database (must already exist and needs to have database creation rights)
- **[Group]**: Group role in Postgres. If left blank, the user name will be used. Postgres prefers a group to be given security rights instead of a user. Users should be assigned to groups (must already exist on the database server).
- **[Password]**: Password for above User
- **[Database]**: Name of the database where the tables need to be created (must already exist)
- **[Schema]**: The name of the schema where the table should be created. PostgreSQL uses public by default. If no value is given for [Schema], public will be used by default. **Important**: The schema must already exist in the database.
- **[SSL]**: A value of true or false will indicate if the connection uses ssl.
- **[Underscore]**: A value of true or false will indicate whether table and column names will be formatted with underscores. Since Postgres doesn't support camel cased names without quotes, all table and column names will be converted to lower case whether or not underscores are used.
- **[Package]**: Name of the package for the generated code. If more than one package is required, two separate files will need to be processed by StreetCRUD, each with a different value for [Package].

The next areas of the text file consist of structs used for code and table generation. The structs are in Go syntax with a few modifications to allow StreetCRUD to generate the proper tables and functions. four keywords appear above the struct to indicate action, table name, file name, and to use prepared SQL statements. Table name and file name can be left blank, causing default names to be used based on the struct name. Additional keywords are used at the end of each line of a struct variable. They indicate what type of column should be created in the database (e.g. [primary] to signal that that column is the primary key). Some of these keywords also cause additional methods to be generated.

#### Keywords Above a Struct for Generating New Code/Table
- **[add struct]**: Indicates that a new struct needs to be processed and added to the database. No text is required next to [add struct].
- **[table]**: A table name can be added next to this or it can be left blank to allow for default naming (e.g., tbl_structName). Table names will be converted to lower case and named using underscores if [Underscore] is set to true. For example, "[table] tblName" will create a new table named tblname or tbl_name depending on the [Underscore setting. If there is only empty space after "[table]" and the name of the defined struct is User, the table name will be tbl_user or tbluser.
- **[file name]**: A file name such as user.go can be added to the right of [file name] which will cause the code-generation file to be named user.go. If no name is given, default naming based on struct name will be used. If multiple struct definitions use the same value for [file name], code will be generated to the same file and not generated in separate files.

#### Keywords Above a Struct for Dealing With an Altered Table/Struct

- **[alter table]**: The name of the table to be altered should appear after the keyword (e.g., [alter table] old_table). This command should be used after changes occur to a previously generated struct. These can include the deletion of a struct variable (dropping a column), altering the order of columns (new struct variable order will mirror new column order), or adding a new struct variable (new column). [alter table] will trigger StreetCRUD to generate new code and a new table. Data will be copied from a previously generated table to the new table. The new table will mirror the new struct, and the data to be copied to the new table will be defined by the user in a series of lines mapping the old column name to the new struct variable name. The order of keywords and struct definition required to follow an [alter table] command can be seen in the example file above, and below they are defined.
- **[copy cols]**: Must appear on the line following [alter table] and before the lines that define how columns are copied from the old table to the new table. This keyword is only used to help improve readability of the user created text file. The mapping of the old column names to the newly defined struct variables should follow this line.
- **OldColName [to] NewStructVarName**: These lines will let StreetCRUD know how data should be copied from the original table to the newly created table (altered table). OldColName is the column name in the existing database table. NewStructVarName is the struct variable name that appears in the new struct. Data from OldColName will be copied to the column that will be created based on the struct variable name. If an OldColName is not mapped, then its data will not be copied to the new table.
- **[add struct]**: Needs to appear before the new struct definition.
- **[table]**: Same as previously defined.
- **[file name]**: Same as above previously defined.
- **[prepared]**: Can be set to true or false. If set to true, then generated code will have prepared sql statements. If false, generated code will have string value sql statements.

#### Struct Keywords
The following keywords can be added to the end of a line that defines a struct variable. There can be 0 to many keywords at the end of each line. These will alter how columns are defined and what methods should be created. Below is an example taken out of a struct definition
~~~
Title string `json:”title"` [index][patch][size:255]
~~~
The three keywords are [index], [patch], and [size:255] all of which are optional and are defined below.
- **[primary]**: This is required and can only appear on one variable. The variable must be one of the variety of int types. This will cause the column to be created with a Postgres sequence. The primary key will auto-increment on insert.
- **[index]**: When used, the column will have an index created which will improve SQL search speeds. I have found that when an index is created, it is usually because a search will be performed using the indexed column. Because of this, an additional method is created that will get all rows where the column value equals a passed in value.
- **[patch]**: Causes a patch (update) method to be created where only the column is updated instead of the entire object. At this time, patch methods generated only support the update of one column, but later, patch-groups will be added to allow patch methods to be created that update more than one column at a time. No keyword is needed for the creation of whole-object updates since those are created by default.
- **[size:n]**: n should be an integer value such as 255. This keyword can be used for string variables to let StreetCRUD know the size of the Postgres "character varying" variable to be created. If [size:n] isn't used, then the database column type will be "character varying" with no size, which is the same as the "text" type.
- **[ignore]**: Used when the variable is of non-basic type, such as struct type. StreetCRUD does not yet support nested non-basic types. A variable column marked with [ignore] will not be added to the database and struct methods.
- **[deleted] and [deletedOn]**: When [deleted] is used, the variable type must be bool. When [deletedOn] is used, the variable type must be time.Time. [deleted] and [deletedOn] can only appear on a single variable in a struct, and they can't be on the same variable. Also, the keywords must appear as a pair. A method will be created that sets the [deleted] column to true and sets the [deletedOn] column to the current date and time.
- **[nulls]**: When used, the column will be set to allow null values. The generated variable will use the "github.com/markbates/going/nulls" package null types because they automatically marshal to and from JSON properly. Supported types are string, int64, float64, bool, []byte, float32, int, int32, uint32, and time.Time. Make sure to run the "go get github.com/markbates/going/nulls" command if this keyword is used. Columns marked as both [deleted] and [nulls] will just be marked as [deleted].

## Table and File Creation Handling
The generated code file(s) will not be formatted, but thanks to goFMT, the code will be perfectly formatted after a save in your text editor of choice is performed.

If a file exists with the same name as a newly generated file, the old file will not be renamed. The generated file will have an incremented number appended.

Along the same lines, if an [alter table] command is performed, the newly generated Go code will not be appended to a previously generated file, but will be added to a new file. This new code may have to be copied and pasted to the previously generated file because it is safe to assume that custom code may have been added to the originally generated file.

If a new table is added and there already exists a table with the same name, the old table will be renamed with an incremented number appended. Tables that are altered will not result in the old table being dropped, but, as stated, they will be renamed. Data will be copied from the old table to the new table according to the column mapping provided by the user if an [alter table] command was executed.

## Gotchas
- At this time, the only type of primary key possible is an auto-incrementing variable of some type of integer.
- Support for nested objects has not yet been added. If your struct has a struct for a variable, use the keyword [ignore] after the variable/column for it to be ignored.
- Fully qualified names for anything database related should not be used. StreetCRUD combines partial elements such as database name, schema, etc., for you.
- StreetCRUD does not make sure that struct variables and columns are unique. Entering identical names in the file to be processed will cause an error. This issue will be addressed in the future.

### Licensed Under the MIT License (see included LICENSE.md file)