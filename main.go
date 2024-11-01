package main

import (
	"fmt"
)


/*
The features of this library are:
1. keeps development team thinking in SQL
2. 100% safe -- never need SQL injection 
3. Very easy to understand
4. Extendable with more helpers
5. 100% cached prepared statements
6. Allows for dynamic where clauses

Advantages:
- When a team is thinking in SQL, they never have to go backwards
- You can add as much convenience sugar to the top of it as you'd like
- At RunPod scale, we need to optimize and review each query
- Prevents the need to cross-consult ORM plus SQL documentation 
- Policy should be to never inject data into SQL

Implemenation:

- There should be a SQL parsing function that will break a qurery up into chunks
    - For example: 
        "SELECT name FROM users WHERE id = $id AND active = true"
            [0] "SELECT name FROM users WHERE id = "
            [1] "$id"
            [2] " AND active = true"
    - Then the odd fields can be ordered properly, looked up in the params, and replaced with ?
    - Same concept goes for {where}, {fields}, {values}, and {fields=values}
    - It is important that the ordering of occurance in the query is mapped to the actual params passed to the sql driver

- Once the SQL is parsed as per above, the final sql + the conenction ID should be used as a key in
  a prepared statement cache.

- When a new query comes in to execute, it should be looked up in the prepared statement cache, and if it is not found,
  it should be prepared and added to the cache, then returned and run.

- I can provide more detail, but this is the essential idea.

*/

func main() {

    // Get a single value.  If it is not found, that is an error.
	value(`SELECT create_date FROM users WHERE id = $id`, param("id", 1234))

    // Get a single value, if it is not found, return nil
    value_nil(`SELECT create_date FROM users WHERE id = $id`, param("id", 12345))
    
    // get a list of statuses
    value_list(`SELECT status FROM statuses WHERE pod_id = $id`, param("id", 1234))

    // Get a single row.  If it is not found, that is an error.
    // take note of the use of $id
	row(`
        SELECT
            id,
            name,
            age,
        FROM 
            users 
        WHERE true
            AND id = $id
        `,
		param("id", 1234),
	)

    // Get a single row, if it is not found, return nil
    row_nil(
        "SELECT id, name, age FROM users WHERE id = $id AND active = $active", 
        param("id", 12345), 
        param("active", true),
    )

    // Take advantage of the {where} automation
    row_list(`
        SELECT
            id,
            name,
            age,
        FROM 
            users   
        WHERE True
            AND NOT deleted
            AND {where}
        `,
        where_gt("age", 30),
        where_lt("age", 40),
    )

    // create a dynamic list of filters to pass along
    filters := []Param{};
    filters = append(filters, where_gt("age", 30))
    filters = append(filters, where_lt("age", 40))
	row_list(`SELECT id, name, age FROM users WHERE {where}`,filters...)

    // Running a complex update with automation for {fields=values} and {where}, as well as params
    execute(`
        UPDATE
            pod
        SET 
            {field=value}
        WHERE true
            AND pod_cat_id = $pod_cat_id
            AND pod_id = (select max(pod_id) from pod_network WHERE category = $category)
            AND NOT deleted
            AND {where}
        `,
        field("active", true),
        field("name", "New Name"),
        field_sql("age", 30, "$age + 1"),
        param("pod_cat_id", 12345),
        param("category", "test"),
        where_not_null("error_message"),
    )

    // automatic insert construction
	insert("user", 
        param("name", "Jason"), 
        param("age", 30),
    )

    // example where we actually just pass actual sql to the update
	update("user", 
        param("name", "New Name"), 
        param_sql("age", nil, "age + 1"),
        where_eq("id", 12345),
        where_eq("active", true),
    )

    // delete a record
    delete("user", param("id", 12345))

}

///////////////////////////////////////////////////////////////////////////////////////

// running SQL that doesn't return a value
func execute(sql string, params ...Param) {
	// print out the sql string
	print_params(sql, params)
}

// getting exactly 1 value, otherwise an error
func value(sql string, params ...Param) {
	// print out the sql string
	print_params(sql, params)
}

// getting one value, if not found, nil
func value_nil(sql string, params ...Param) {
    // print out the sql string
    print_params(sql, params)
}

// getting a list or array of values
func value_list(sql string, params ...Param) {
	print_params(sql, params)
}

// getting a single row from the database, or an error if it is not found
func row(sql string, params ...Param) {
	// print out the sql string
	print_params(sql, params)
}

// getting a single row from the database, or nil if it is not found
func row_nil(sql string, params ...Param) {
    // print out the sql string
    print_params(sql, params)
}

// getting a list of rows from the database
func row_list(sql string, params ...Param) {
	print_params(sql, params)
}

// constructing and executing an insert statement
// ideally, we should only accept params of type=field at compile time (maybe a different struct?)
func insert(table string, params ...Param) {
	// print out the sql string
	sql := fmt.Sprintf("INSERT INTO %s  ({fields}) VALUES ({values})", quote_ident(table))
	execute(sql, params...)
}

// construct and execute an update statement
// ideally, we should only accept params of type=field or type=where at compile time (maybe a different struct?)
func update(table string, params ...Param) {
	// print out the sql string
	sql := fmt.Sprintf("UPDATE %s SET {fields=values} WHERE {where}", quote_ident(table))
	execute(sql, params...)
}

// construct and execute a delete statement
// ideally, we should only accept params of type=where at compile time (maybe a different struct?)
func delete(table string, params ...Param) {
	// print out the sql string
	sql := fmt.Sprintf("DELETE FROM %s WHERE {where}", quote_ident(table))
	execute(sql, params...)
}

///////////////////////////////////////////////////////////////////////////////////////

// need a function to properly quote field or table names (this is not it)
func quote_ident(name string) string {
	return "`" + name + "`"
}

// / In rust I would use an enum to differentiate between the different types of params
// / Not sure best way in go, but this can either be a field value, or a where condition
type Param struct {
	Type  string
	Field string
	Value interface{}
	Sql   string
}

func param(field string, value interface{}) Param {
	return Param{Type: "param", Field: field, Value: value, }
}

func param_sql(field string, value interface{}, value_sql string) Param {
	return Param{Type: "param_sql", Field: field, Value: value, Sql: value_sql}
}

func field(field string, value interface{}) Param {
    sql := fmt.Sprintf("$%s", field)
    return Param{Type: "field", Field: field, Value: value, Sql: sql}
}

func field_sql(field string, value interface{}, sql string) Param {
    return Param{Type: "field_sql", Field: field, Value: value, Sql: sql}
}

func where_null(field string) Param {

	return Param{Type: "where_null", Field: field, Value: nil, Sql: ""}
}

func where_not_null(field string) Param {
	return Param{Type: "where_not_null", Field: field, Value: nil, Sql: ""}
}

func where_eq(field string, value interface{}) Param {
	return Param{Type: "where_eq", Field: field, Value: value, Sql: ""}
}

func where_ne(field string, value interface{}) Param {
	return Param{Type: "where_ne", Field: field, Value: value, Sql: ""}
}

func where_gt(field string, value interface{}) Param {
	return Param{Type: "where_gt", Field: field, Value: value, Sql: ""}
}

func where_gte(field string, value interface{}) Param {
	return Param{Type: "where_gte", Field: field, Value: value, Sql: ""}
}

func where_lt(field string, value interface{}) Param {
	return Param{Type: "where_lt", Field: field, Value: value, Sql: ""}
}

func where_lte(field string, value interface{}) Param {
	return Param{Type: "where_lte", Field: field, Value: value, Sql: ""}
}


func print_params(sql string, params []Param) {
	fmt.Println("-------------------------------------------")

	// print the sql string
	fmt.Printf("%s\n", sql)

    // Print out the fields
    for _, param := range params {
        switch param.Type {
            case "field":
                fmt.Printf("  %s = %v\n", param.Field, param.Value)
            case "field_sql":
                fmt.Printf("  %s = %s\n", param.Field, param.Sql)
        }
    }

	// print out the params
	for _, param := range params {
		switch param.Type {
		case "param":
			fmt.Printf("  %s = %v\n", param.Field, param.Value)
		case "param_sql":
			fmt.Printf("  %s = %s\n", param.Field, param.Sql)
		}
	}

	for _, param := range params {
		switch param.Type {
		case "where_null":
			fmt.Printf("  WHERE %s IS NULL\n", param.Field)
		case "where_not_null":
			fmt.Printf("  WHERE %s IS NOT NULL\n", param.Field)
		case "where_eq":
			fmt.Printf("  WHERE %s = %v\n", param.Field, param.Value)
		case "where_ne":
			fmt.Printf("  WHERE %s != %v\n", param.Field, param.Value)
		case "where_gt":
			fmt.Printf("  WHERE %s > %v\n", param.Field, param.Value)
		case "where_gte":
			fmt.Printf("  WHERE %s >= %v\n", param.Field, param.Value)
		case "where_lt":
			fmt.Printf("  WHERE %s < %v\n", param.Field, param.Value)
		case "where_lte":
			fmt.Printf("  WHERE %s <= %v\n", param.Field, param.Value)
		}
	}

}
