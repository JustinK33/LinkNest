class AddPerformanceIndexes < ActiveRecord::Migration[8.1]
  def change
    # Index for unique visitor queries (used in link analytics)
    add_index :link_clicks, :ip_address, name: "index_link_clicks_on_ip_address"

    # Composite index for top links queries by user and click count
    add_index :links, [ :user_id, :click_count ], name: "index_links_on_user_id_and_click_count"

    # Index for link clicks date range queries with IP (for unique visitor calculations)
    add_index :link_clicks, [ :link_id, :created_at, :ip_address ], name: "index_link_clicks_on_link_date_ip"
  end
end
