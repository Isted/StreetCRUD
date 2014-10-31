# Street CRUD

StreetCRUD is a code generating command line utility for people who aren't fans of ORMs, but like a kickstart creating struct methods, tables, and queries for basic CRUD functionality. You only have to supply your structs, database info, and a few keywords in a text file that StreetCRUD will process. This allows the programmer to add methods and queries at a later date without having to wrestle with an ORM. You keep all the power, but don't have to start at level one. At this time, StreetCRUD uses postgreSQL, JSON, and golang. As a nice side benefit StreetCRUD can be used to easily reorder columns in a Postgres tables.

Features Include:
*Table creation (if doesn't exist)
*Insert, Update (Patch optional), and Delete Methods and Queries Created
*StreetCRUD Can Be Rerun to alter methods and queries if there is a struct change
*Data is safely copied via a map if a struct is altered
*Methods Return and Receive JSON
*Reordering of table columns

##Getting Started

~~~
package main

import "fmt"

~~~

###Licensed Under the MIT License (see included LICENSE.md file)

