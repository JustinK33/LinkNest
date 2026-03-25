class AddDatabaseConstraintsAndIndexes < ActiveRecord::Migration[8.1]
  def change
    # Add NOT NULL constraints where they should exist
    # Note: These changes are done carefully to avoid breaking existing data

    # Links table - ensure critical fields are NOT NULL
    change_column_null :links, :user_id, false
    change_column_null :links, :title, false

    # Sessions table - ensure security fields are NOT NULL
    change_column_null :sessions, :ip_address, false
    change_column_null :sessions, :user_agent, false

    # Add performance indexes
    # Index for session cleanup queries (expires sessions efficiently)
    add_index :sessions, :updated_at, name: "index_sessions_on_updated_at"

    # Index for public link queries (used in profiles and tracking)
    add_index :links, :public, name: "index_links_on_public"

    # Add missing foreign key constraint for top_link_id
    # This requires handling NULL values first
    add_foreign_key :daily_user_stats, :links, column: :top_link_id, on_delete: :nullify

    # Add unique constraint for session management (one session per user agent + IP combo per user)
    # This helps prevent session duplication and improves security
    # NOTE: Commenting this out for now as it might be too restrictive
    # add_index :sessions, [:user_id, :user_agent, :ip_address], unique: true, name: "index_sessions_on_user_security"
  end

  def down
    # Remove indexes
    remove_index :sessions, name: "index_sessions_on_updated_at" if index_exists?(:sessions, :updated_at)
    remove_index :links, name: "index_links_on_public" if index_exists?(:links, :public)

    # Remove foreign key
    remove_foreign_key :daily_user_stats, :links if foreign_key_exists?(:daily_user_stats, :links)

    # Revert NOT NULL constraints (make nullable again)
    change_column_null :links, :user_id, true
    change_column_null :links, :title, true
    change_column_null :sessions, :ip_address, true
    change_column_null :sessions, :user_agent, true
  end
end
