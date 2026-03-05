class CreateHourlyLinkStats < ActiveRecord::Migration[8.1]
  def change
    create_table :hourly_link_stats do |t|
      t.references :user, null: false, foreign_key: true
      t.references :link, null: false, foreign_key: true
      t.datetime :hour, null: false  # Start of the hour (e.g., 2026-03-05 14:00:00)
      t.integer :click_count, default: 0
      t.integer :unique_visitors, default: 0

      t.timestamps
    end

    # Indexes for dashboard queries
    add_index :hourly_link_stats, [ :user_id, :hour ]
    add_index :hourly_link_stats, :hour

    # Unique constraint: one row per link per hour
    add_index :hourly_link_stats, [ :link_id, :hour ], unique: true
  end
end
