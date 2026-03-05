class FinalizeAnalyticsSchema < ActiveRecord::Migration[8.1]
  def change
    # Link clicks table - raw event stream
    unless table_exists?(:link_clicks)
      create_table :link_clicks do |t|
        t.references :user, null: false, foreign_key: true
        t.references :link, null: false, foreign_key: true
        t.string :referrer
        t.string :user_agent
        t.string :ip_address
        t.string :country_code
        t.string :device_type
        t.string :browser_name
        t.timestamps
      end
      add_index :link_clicks, [ :user_id, :created_at ] unless index_exists?(:link_clicks, [ :user_id, :created_at ])
      add_index :link_clicks, [ :link_id, :created_at ] unless index_exists?(:link_clicks, [ :link_id, :created_at ])
      add_index :link_clicks, :created_at unless index_exists?(:link_clicks, :created_at)
    end

    # Hourly aggregated stats
    unless table_exists?(:hourly_link_stats)
      create_table :hourly_link_stats do |t|
        t.references :user, null: false, foreign_key: true
        t.references :link, null: false, foreign_key: true
        t.datetime :hour, null: false
        t.integer :click_count, default: 0
        t.integer :unique_visitors, default: 0
        t.timestamps
      end
      add_index :hourly_link_stats, [ :user_id, :hour ] unless index_exists?(:hourly_link_stats, [ :user_id, :hour ])
      add_index :hourly_link_stats, :hour unless index_exists?(:hourly_link_stats, :hour)
      add_index :hourly_link_stats, [ :link_id, :hour ], unique: true unless index_exists?(:hourly_link_stats, [ :link_id, :hour ])
    end

    # Daily aggregated stats
    unless table_exists?(:daily_user_stats)
      create_table :daily_user_stats do |t|
        t.references :user, null: false, foreign_key: true
        t.date :date, null: false
        t.integer :total_clicks, default: 0
        t.integer :unique_visitors, default: 0
        t.integer :top_link_id
        t.integer :top_link_clicks, default: 0
        t.timestamps
      end
      add_index :daily_user_stats, :date unless index_exists?(:daily_user_stats, :date)
      add_index :daily_user_stats, [ :user_id, :date ], unique: true unless index_exists?(:daily_user_stats, [ :user_id, :date ])
    end
  end
end
