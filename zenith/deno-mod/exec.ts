// deno-lint-ignore-file no-explicit-any
import { GraphQLNonNull, GraphQLScalarType } from "npm:graphql";
import { execute } from "./deps.ts";
import { hello, add } from "./src/queries.ts";
import { schema } from "./src/schema.ts";

const result = await execute({
  schema,
  document: hello,
  variableValues: {
    name: "Tsiry",
  },
  contextValue: {
    x: 1,
  },
});

console.log(result.data);

const getReturnType = (schema: any, queryName: string) => {
  const queryType = schema.getQueryType();
  const queryField = queryType?.getFields()[queryName];
  const queryFieldType =
    (queryField?.type as GraphQLScalarType).name ||
    (queryField?.type as GraphQLNonNull<GraphQLScalarType>).ofType?.name;
  return queryFieldType;
};

const getArgsType = (schema: any, queryName: string) => {
  const queryType = schema.getQueryType();
  const queryField = queryType?.getFields()[queryName];
  return queryField?.args.map((arg: any) => {
    const argType =
      (arg.type as GraphQLScalarType).name ||
      (arg.type as GraphQLNonNull<GraphQLScalarType>).ofType?.name;
    return { name: arg.name, type: argType };
  });
};

const helloType = getReturnType(schema, "hello");
const helloArgs = getArgsType(schema, "hello");
const addType = getReturnType(schema, "add");
const addArgs = getArgsType(schema, "add");
const isEvenType = getReturnType(schema, "isEven");
const isEvenArgs = getArgsType(schema, "isEven");

console.log(helloType);
console.log(helloArgs);
console.log(addType);
console.log(addArgs);
console.log(isEvenType);
console.log(isEvenArgs);

const sum = await execute({
  schema,
  document: add,
  variableValues: {
    a: 1,
    b: 2,
  },
});

console.log(sum.data);
