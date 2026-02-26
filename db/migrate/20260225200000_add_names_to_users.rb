class AddNamesToUsers < ActiveRecord::Migration[8.1]
  def change
    add_column :users, :first_name, :string, null: false, default: "Unknown"
    add_column :users, :last_name, :string, null: false, default: "User"
  end
end
