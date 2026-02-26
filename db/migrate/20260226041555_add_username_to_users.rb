class AddUsernameToUsers < ActiveRecord::Migration[8.0]
  def change
    add_index :users, :username, unique: true
    change_column_default :users, :username, from: "Random", to: nil
  end
end
