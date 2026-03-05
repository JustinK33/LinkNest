class AddProfileFieldsToUsers < ActiveRecord::Migration[8.1]
  def change
    add_column :users, :slug, :string, null: false
    add_column :users, :bio, :text
    add_column :users, :avatar_url, :string
    add_column :users, :profile_color, :string, default: "#3b82f6"

    # Indexes
    add_index :users, :slug, unique: true
  end
end
