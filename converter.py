import sqlite3

# This script updates tables to use correct datetime formatting so that sqlite
# can understand them as dates.


def update_columns(conn, cur, table_name):
    if table_name == "budget":
        recordname = "purchasedate"
    elif table_name == "salary":
        recordname = "recordtime"
    else:
        raise "No recordname"

    res = cur.execute(f"SELECT id, {recordname} FROM {table_name}")

    fixed_dates = list()
    for row in res.fetchall():
        s = row[recordname].split("-")
        if len(s[1]) == 4:
            year = s[1]
            month = s[0]
        else:
            year = s[0]
            month = s[1]
        fixed_dates.append((row["id"], f"{year}-{month}-01"))
    for idx, date in fixed_dates:
        cur.execute(f"UPDATE {table_name} SET {recordname} = ? WHERE id=?;", [date, idx])
    conn.commit()


def main():
    conn = sqlite3.connect("budget.db")
    conn.row_factory = sqlite3.Row
    cur = conn.cursor()

    # Backup the db, perform modifications, check the results
    update_columns(conn, cur, "budget")
    update_columns(conn, cur, "salary")


if __name__ == '__main__':
    main()
