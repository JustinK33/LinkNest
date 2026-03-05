class CreateDailyUserStats < ActiveRecord::Migration[8.1]
  def change
    unless table_exists?(:daily_user_stats)
      create_table :daily_user_stats do |t|
        t.references :user, null: false, foreign_key: true
        t.date :date, null: false  # The date (e.g., 2026-03-05)
        t.integer :total_clicks, default: 0
        t.integer :unique_visitors, default: 0
        t.integer :top_link_id  # Link with most clicks on this day
        t.integer :top_link_clicks, default: 0

        t.timestamps
      end

      # Indexes for trend queries
      add_index :daily_user_stats, [ :user_id, :date ]
      add_index :daily_user_stats, :date

      # Unique constraint: one row per user per day
      add_index :daily_user_stats, [ :user_id, :date ], unique: true
    end
  end
end
