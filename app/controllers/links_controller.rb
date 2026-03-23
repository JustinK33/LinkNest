class LinksController < ApplicationController
  before_action :set_link, only: [ :edit, :update, :destroy ]

  def index
    redirect_to dashboard_path
  end

  def new
    @link = Current.session.user.links.new
    @entry_mode = requested_entry_mode
  end

  def edit
    @entry_mode = requested_entry_mode(default: @link.resume_pdf.attached? ? "file" : "link")
  end

  def create
    @entry_mode = requested_entry_mode
    @link = Current.session.user.links.new(link_params)
    @link.position = next_position

    if @link.save
      sync_resume_url_if_needed(@link)
      redirect_to dashboard_path, notice: "Link created successfully!"
    else
      render :new, status: :unprocessable_entity
    end
  end

  def update
    @entry_mode = requested_entry_mode(default: @link.resume_pdf.attached? ? "file" : "link")
    purge_resume_pdf_if_requested(@link)

    if @link.update(link_params)
      sync_resume_url_if_needed(@link)
      redirect_to dashboard_path, notice: "Link updated successfully!"
    else
      render :edit, status: :unprocessable_entity
    end
  end

  def destroy
    @link.destroy
    redirect_to dashboard_path, notice: "Link deleted successfully!"
  end

  private

    def set_link
      @link = Current.session.user.links.find(params[:id])
    end

    def link_params
      params.expect(link: [ :title, :url, :resume_pdf ])
    end

    def requested_entry_mode(default: "link")
      mode = params[:entry_mode] || params[:mode]
      return mode if %w[link file].include?(mode)

      default
    end

    def next_position
      (Current.session.user.links.maximum(:position) || -1) + 1
    end

    def sync_resume_url_if_needed(link)
      return unless link.resume_pdf.attached?

      link.update_column(:url, rails_blob_path(link.resume_pdf, only_path: true))
    end

    def purge_resume_pdf_if_requested(link)
      return unless params.dig(:link, :remove_resume_pdf) == "1"

      link.resume_pdf.purge_later if link.resume_pdf.attached?
      link.update_column(:url, "")
    end
end
