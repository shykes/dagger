import { gql } from "./deps.ts";

export const hello = gql`
  query Hello($name: String!) {
    hello(name: $name)
  }
`;
