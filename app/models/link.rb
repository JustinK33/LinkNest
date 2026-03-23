class Link < ApplicationRecord
  belongs_to :user
  has_many :link_clicks, dependent: :destroy
  has_many :hourly_link_stats, dependent: :destroy
  has_one_attached :resume_pdf

  validates :title, presence: true
  validate :url_or_resume_present
  validate :url_format_is_valid
  validate :resume_pdf_must_be_pdf
  validates :user_id, presence: true
  validates :position, presence: true, numericality: { only_integer: true }

  scope :ordered, -> { order(:position) }
  scope :active, -> { where(deleted_at: nil) }

  # Update click count from aggregated stats
  def update_click_count!
    total_clicks = link_clicks.count
    update(click_count: total_clicks)
  end

  # Get clicks for a time period
  def clicks_in_range(start_time, end_time)
    link_clicks.where(created_at: start_time..end_time)
  end

  # Get unique visitors for a time period
  def unique_visitors_in_range(start_time, end_time)
    link_clicks.where(created_at: start_time..end_time).distinct.count(:ip_address)
  end

  private
    def url_or_resume_present
      return if url.present? || resume_pdf.attached?

      errors.add(:base, "Provide a URL or upload a PDF resume")
    end

    def url_format_is_valid
      return if url.blank?
      return if url.start_with?("/")
      return if url.start_with?("mailto:", "tel:")
      return if url.match?(URI::DEFAULT_PARSER.make_regexp)

      errors.add(:url, "must be a valid URL")
    end

    def resume_pdf_must_be_pdf
      return unless resume_pdf.attached?
      return if resume_pdf.content_type == "application/pdf"

      errors.add(:resume_pdf, "must be a PDF file")
    end
end
