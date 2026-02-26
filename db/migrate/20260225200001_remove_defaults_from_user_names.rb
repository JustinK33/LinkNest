class RemoveDefaultsFromUserNames < ActiveRecord::Migration[8.1]
  def change
    change_column_default :users, :username, from: "Random", to: nil
    change_column_default :users, :first_name, from: "Unknown", to: nil
    change_column_default :users, :last_name, from: "User", to: nil
  end
end
