class CreateLinkClicks < ActiveRecord::Migration[8.1]
  def change
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

    # Critical indexes for analytics queries (add_reference creates indexes on foreign keys)
    add_index :link_clicks, [ :user_id, :created_at ]
    add_index :link_clicks, [ :link_id, :created_at ]
    add_index :link_clicks, :created_at  # For time-range queries
  end
end
