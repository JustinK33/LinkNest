class AddPublicToLinks < ActiveRecord::Migration[7.1]
  def change
    add_column :links, :public, :boolean, default: true, null: false
    add_index :links, [ :user_id, :public, :position ]
  end
end
