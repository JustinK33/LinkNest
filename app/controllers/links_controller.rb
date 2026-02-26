class LinksController < ApplicationController
  def index
    redirect_to dashboard_path
  end

  def new
    # Mock link object for form
    @link = { title: "", url: "", visible: true }
  end

  def edit
    # Mock link data
    @link = { id: params[:id], title: "My Portfolio", url: "https://example.com/portfolio", visible: true, position: 1 }
  end

  def create
    # This will be implemented with actual model logic later
    redirect_to dashboard_path, notice: "Link created successfully!"
  end

  def update
    # This will be implemented with actual model logic later
    redirect_to dashboard_path, notice: "Link updated successfully!"
  end

  def destroy
    # This will be implemented with actual model logic later
    redirect_to dashboard_path, notice: "Link deleted successfully!"
  end
end
