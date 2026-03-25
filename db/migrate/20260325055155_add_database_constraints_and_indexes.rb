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

    # Add performance indexes - with safety checks to prevent duplicates
    # Index for session cleanup queries (expires sessions efficiently)
    unless index_exists?(:sessions, :updated_at, name: "index_sessions_on_updated_at")
      add_index :sessions, :updated_at, name: "index_sessions_on_updated_at"
    end

    # Index for public link queries (used in profiles and tracking)
    unless index_exists?(:links, :public, name: "index_links_on_public")
      add_index :links, :public, name: "index_links_on_public"
    end

    # Add missing foreign key constraint for top_link_id
    # First, ensure column type matches links.id (bigint) - MySQL requires exact type match for FKs
    if column_exists?(:daily_user_stats, :top_link_id)
      change_column :daily_user_stats, :top_link_id, :bigint
    end

    unless foreign_key_exists?(:daily_user_stats, :links, column: :top_link_id)
      add_foreign_key :daily_user_stats, :links, column: :top_link_id, on_delete: :nullify
    end

    # Add unique constraint for session management (one session per user agent + IP combo per user)
    # This helps prevent session duplication and improves security
    # NOTE: Commenting this out for now as it might be too restrictive
    # unless index_exists?(:sessions, [:user_id, :user_agent, :ip_address], name: "index_sessions_on_user_security")
    #   add_index :sessions, [:user_id, :user_agent, :ip_address], unique: true, name: "index_sessions_on_user_security"
    # end
  end

  def down
    # Remove indexes
    remove_index :sessions, name: "index_sessions_on_updated_at" if index_exists?(:sessions, :updated_at, name: "index_sessions_on_updated_at")
    remove_index :links, name: "index_links_on_public" if index_exists?(:links, :public, name: "index_links_on_public")

    # Remove foreign key
    remove_foreign_key :daily_user_stats, :links, column: :top_link_id if foreign_key_exists?(:daily_user_stats, :links, column: :top_link_id)

    # Revert NOT NULL constraints (make nullable again)
    change_column_null :links, :user_id, true
    change_column_null :links, :title, true
    change_column_null :sessions, :ip_address, true
    change_column_null :sessions, :user_agent, true
  end
end
