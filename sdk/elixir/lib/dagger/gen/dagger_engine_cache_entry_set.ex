# This file generated by `dagger_codegen`. Please DO NOT EDIT.
defmodule Dagger.DaggerEngineCacheEntrySet do
  @moduledoc "A set of cache entries returned by a query to a cache"

  alias Dagger.Core.Client
  alias Dagger.Core.QueryBuilder, as: QB

  @derive Dagger.ID

  defstruct [:query_builder, :client]

  @type t() :: %__MODULE__{}

  @doc "The total disk space used by the cache entries in this set."
  @spec disk_space_bytes(t()) :: {:ok, integer()} | {:error, term()}
  def disk_space_bytes(%__MODULE__{} = dagger_engine_cache_entry_set) do
    query_builder =
      dagger_engine_cache_entry_set.query_builder |> QB.select("diskSpaceBytes")

    Client.execute(dagger_engine_cache_entry_set.client, query_builder)
  end

  @doc "The list of individual cache entries in the set"
  @spec entries(t()) :: {:ok, [Dagger.DaggerEngineCacheEntry.t()]} | {:error, term()}
  def entries(%__MODULE__{} = dagger_engine_cache_entry_set) do
    query_builder =
      dagger_engine_cache_entry_set.query_builder |> QB.select("entries") |> QB.select("id")

    with {:ok, items} <- Client.execute(dagger_engine_cache_entry_set.client, query_builder) do
      {:ok,
       for %{"id" => id} <- items do
         %Dagger.DaggerEngineCacheEntry{
           query_builder:
             QB.query()
             |> QB.select("loadDaggerEngineCacheEntryFromID")
             |> QB.put_arg("id", id),
           client: dagger_engine_cache_entry_set.client
         }
       end}
    end
  end

  @doc "The number of cache entries in this set."
  @spec entry_count(t()) :: {:ok, integer()} | {:error, term()}
  def entry_count(%__MODULE__{} = dagger_engine_cache_entry_set) do
    query_builder =
      dagger_engine_cache_entry_set.query_builder |> QB.select("entryCount")

    Client.execute(dagger_engine_cache_entry_set.client, query_builder)
  end

  @doc "A unique identifier for this DaggerEngineCacheEntrySet."
  @spec id(t()) :: {:ok, Dagger.DaggerEngineCacheEntrySetID.t()} | {:error, term()}
  def id(%__MODULE__{} = dagger_engine_cache_entry_set) do
    query_builder =
      dagger_engine_cache_entry_set.query_builder |> QB.select("id")

    Client.execute(dagger_engine_cache_entry_set.client, query_builder)
  end
end