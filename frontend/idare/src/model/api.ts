export enum QUERY_COLUMN_TYPES {
  NUMBER = 1,
  STRING = 10,
  BOOL = 20,
  JSON_NUMBER = 30,
  JSON_STRING = 40,
  ASSOCIATION_NUMBER = 50,
  ASSOCIATION_STRING = 60,
  IN_INT = 70,
  IN_STRING = 80,
  NULL = 90,
  NOT_NULL = 91,
}

export enum SORT_COLUMN_TYPES {
  NORMAL = 1,
  JSONB = 10,
  ASSOCIATION = 20,
}

export interface IApiQueryPagination extends IApiQuery, IApiPagination {
}

export interface IApiQuery {
  query: string[]
  columns: string[]
  columnTypes: QUERY_COLUMN_TYPES[]
}

export class ApiQuery implements IApiQuery {
  query: string[]
  columns: string[]
  columnTypes: QUERY_COLUMN_TYPES[]

  constructor(query?: [], columns?: [], columnTypes?: []) {
    this.query = query ?? []
    this.columns = columns ?? []
    this.columnTypes = columnTypes ?? []
  }

  append(query: string, column: string, columnType?: QUERY_COLUMN_TYPES) {
    for (let i = 0; i < this.columns.length; i++) {
      if (this.columns[i] === column) {
        this.query[i] = query
        this.columns[i] = column
        this.columnTypes[i] = columnType ?? QUERY_COLUMN_TYPES.STRING
        if (!query) {
          this.query.splice(i, 1)
          this.columns.splice(i, 1)
          this.columnTypes.splice(i, 1)
        }

        return
      }
    }
    if (!query)
      return

    this.query.push(query)
    this.columns.push(column)
    this.columnTypes.push(columnType ?? QUERY_COLUMN_TYPES.STRING)
  }
}

export interface IApiPagination {
  page: number
  perPage: number
  sortColumns: string[]
  sortColumnTypes: SORT_COLUMN_TYPES[]
  sortOrders: string[]
}
