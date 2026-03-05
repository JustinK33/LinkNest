class RenameProductsToLinksAndAddUserAssociation < ActiveRecord::Migration[8.1]
  def change
    # Rename the products table to links
    rename_table :products, :links

    # Update the existing foreign key from subscribers
    rename_column :subscribers, :product_id, :link_id

    # Add user_id foreign key to links
    add_reference :links, :user, foreign_key: true

    # Add analytics columns
    add_column :links, :url, :string
    add_column :links, :click_count, :integer, default: 0
    add_column :links, :position, :integer, default: 0
    add_column :links, :icon_color, :string, default: "#3b82f6"

    # Rename name to title for consistency
    rename_column :links, :name, :title

    # Add indexes for performance (add_reference already creates user_id index)
    add_index :links, [ :user_id, :position ]
    add_index :links, [ :user_id, :created_at ]
  end
end
